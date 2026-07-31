package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gowoobro/global/setting"

	"github.com/gofiber/fiber/v2"
)

// seedBlocklist 는 DB 를 거치지 않고 캐시를 직접 채운다. TTL 안쪽이면 Entries 가
// DB 를 읽지 않으므로 테스트가 DB 없이 돈다.
func seedBlocklist(t *testing.T, rules map[string]string) {
	t.Helper()

	var entries []blockRule
	for address, reason := range rules {
		entry, err := setting.NewIP(address)
		if err != nil {
			t.Fatalf("setting.NewIP(%q) failed: %v", address, err)
		}
		entries = append(entries, blockRule{IP: entry, Reason: reason})
	}

	blocklist.mutex.Lock()
	blocklist.entries = entries
	blocklist.loadedAt = time.Now()
	blocklist.loaded = true
	blocklist.mutex.Unlock()
}

func newGuardedApp() *fiber.App {
	app := fiber.New()
	app.Use(IpBlockGuard)
	app.Get("/api/projects", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
	return app
}

func TestIpBlockGuard(t *testing.T) {
	tests := []struct {
		name    string
		rules   map[string]string
		headers map[string]string
		want    int
	}{
		{
			name:    "차단 목록에 없으면 통과",
			rules:   map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{"X-Real-IP": "1.1.1.1"},
			want:    http.StatusOK,
		},
		{
			name:    "단일 주소 일치하면 차단",
			rules:   map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{"X-Real-IP": "8.8.8.8"},
			want:    http.StatusForbidden,
		},
		{
			name:    "CIDR 대역 안이면 차단",
			rules:   map[string]string{"10.0.0.0/8": "차단됨"},
			headers: map[string]string{"X-Real-IP": "10.1.2.3"},
			want:    http.StatusForbidden,
		},
		{
			name:    "CIDR 대역 밖이면 통과",
			rules:   map[string]string{"10.0.0.0/8": "차단됨"},
			headers: map[string]string{"X-Real-IP": "11.1.2.3"},
			want:    http.StatusOK,
		},
		{
			name:    "범위 표기 안이면 차단",
			rules:   map[string]string{"192.168.1.10-20": "차단됨"},
			headers: map[string]string{"X-Real-IP": "192.168.1.15"},
			want:    http.StatusForbidden,
		},
		{
			name:    "범위 표기 밖이면 통과",
			rules:   map[string]string{"192.168.1.10-20": "차단됨"},
			headers: map[string]string{"X-Real-IP": "192.168.1.25"},
			want:    http.StatusOK,
		},
		{
			// 프록시가 덮어쓰는 X-Real-IP 가 우선이라 XFF 위조로는 남을 차단시킬 수 없다.
			name:  "위조된 X-Forwarded-For 는 X-Real-IP 를 못 이긴다",
			rules: map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{
				"X-Forwarded-For": "8.8.8.8",
				"X-Real-IP":       "1.1.1.1",
			},
			want: http.StatusOK,
		},
		{
			// 반대로 차단 대상이 XFF 를 심어 자기 자신을 숨기려 해도 소용없다.
			name:  "X-Real-IP 가 차단 대상이면 XFF 로 우회 불가",
			rules: map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1",
				"X-Real-IP":       "8.8.8.8",
			},
			want: http.StatusForbidden,
		},
		{
			name:    "X-Real-IP 가 없으면 XFF 첫 값으로 판정",
			rules:   map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{"X-Forwarded-For": "8.8.8.8, 172.18.0.5"},
			want:    http.StatusForbidden,
		},
		{
			name:    "차단 목록이 비어있으면 통과",
			rules:   nil,
			headers: map[string]string{"X-Real-IP": "8.8.8.8"},
			want:    http.StatusOK,
		},
		{
			// setting.NewIP 는 IPv4 전용이다. 판정 불가는 통과시킨다(fail open).
			name:    "판정할 수 없는 주소는 통과",
			rules:   map[string]string{"8.8.8.8": "차단됨"},
			headers: map[string]string{"X-Real-IP": "2001:db8::1"},
			want:    http.StatusOK,
		},
	}

	app := newGuardedApp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedBlocklist(t, tt.rules)

			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.want {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.want)
			}
		})
	}
}

// 차단 사유는 막힌 방문자에게 그대로 보여주므로 응답 본문에 실려야 한다.
func TestIpBlockGuardReturnsReason(t *testing.T) {
	tests := []struct {
		name       string
		reason     string
		wantReason string
	}{
		{
			name:       "사유가 있으면 그대로 내려준다",
			reason:     "스팸 질문 반복 등록",
			wantReason: "스팸 질문 반복 등록",
		},
		{
			// 프론트가 기존 일반 오류 문구로 대체할 수 있도록 빈 값을 그대로 내려준다.
			name:       "사유가 비어 있으면 빈 문자열",
			reason:     "",
			wantReason: "",
		},
	}

	app := newGuardedApp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedBlocklist(t, map[string]string{"8.8.8.8": tt.reason})

			req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
			req.Header.Set("X-Real-IP", "8.8.8.8")

			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusForbidden {
				t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusForbidden)
			}

			var body struct {
				Code    string `json:"code"`
				Message string `json:"message"`
				Reason  string `json:"reason"`
			}
			if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
				t.Fatalf("decode body failed: %v", err)
			}

			if body.Reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", body.Reason, tt.wantReason)
			}
			if body.Code != "error" {
				t.Errorf("code = %q, want %q", body.Code, "error")
			}
		})
	}
}

func TestClientIPPrefersRealIP(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    string
	}{
		{
			name:    "X-Real-IP 우선",
			headers: map[string]string{"X-Real-IP": "1.1.1.1", "X-Forwarded-For": "2.2.2.2"},
			want:    "1.1.1.1",
		},
		{
			name:    "X-Real-IP 없으면 XFF 첫 값",
			headers: map[string]string{"X-Forwarded-For": "2.2.2.2, 172.18.0.5"},
			want:    "2.2.2.2",
		},
		{
			name:    "공백은 잘라낸다",
			headers: map[string]string{"X-Forwarded-For": "  2.2.2.2  , 172.18.0.5"},
			want:    "2.2.2.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string

			app := fiber.New()
			app.Get("/", func(c *fiber.Ctx) error {
				got = clientIP(c)
				return c.SendString("ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			res, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			defer res.Body.Close()

			if got != tt.want {
				t.Errorf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

// seedEmptyBlocklist 는 차단 목록을 비운 상태로 캐시를 채운다. 인증 테스트가
// DB 를 타지 않게 하려는 용도다.
func seedEmptyBlocklist() {
	blocklist.mutex.Lock()
	blocklist.entries = nil
	blocklist.loadedAt = time.Now()
	blocklist.loaded = true
	blocklist.mutex.Unlock()
}
