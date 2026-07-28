package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gowoobro/global/setting"

	"github.com/gofiber/fiber/v2"
)

// seedBlocklist 는 DB 를 거치지 않고 캐시를 직접 채운다. TTL 안쪽이면 Entries 가
// DB 를 읽지 않으므로 테스트가 DB 없이 돈다.
func seedBlocklist(t *testing.T, addresses ...string) {
	t.Helper()

	var entries []*setting.IP
	for _, address := range addresses {
		entry, err := setting.NewIP(address)
		if err != nil {
			t.Fatalf("setting.NewIP(%q) failed: %v", address, err)
		}
		entries = append(entries, entry)
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
		rules   []string
		headers map[string]string
		want    int
	}{
		{
			name:    "차단 목록에 없으면 통과",
			rules:   []string{"8.8.8.8"},
			headers: map[string]string{"X-Real-IP": "1.1.1.1"},
			want:    http.StatusOK,
		},
		{
			name:    "단일 주소 일치하면 차단",
			rules:   []string{"8.8.8.8"},
			headers: map[string]string{"X-Real-IP": "8.8.8.8"},
			want:    http.StatusForbidden,
		},
		{
			name:    "CIDR 대역 안이면 차단",
			rules:   []string{"10.0.0.0/8"},
			headers: map[string]string{"X-Real-IP": "10.1.2.3"},
			want:    http.StatusForbidden,
		},
		{
			name:    "CIDR 대역 밖이면 통과",
			rules:   []string{"10.0.0.0/8"},
			headers: map[string]string{"X-Real-IP": "11.1.2.3"},
			want:    http.StatusOK,
		},
		{
			name:    "범위 표기 안이면 차단",
			rules:   []string{"192.168.1.10-20"},
			headers: map[string]string{"X-Real-IP": "192.168.1.15"},
			want:    http.StatusForbidden,
		},
		{
			name:    "범위 표기 밖이면 통과",
			rules:   []string{"192.168.1.10-20"},
			headers: map[string]string{"X-Real-IP": "192.168.1.25"},
			want:    http.StatusOK,
		},
		{
			// 프록시가 덮어쓰는 X-Real-IP 가 우선이라 XFF 위조로는 남을 차단시킬 수 없다.
			name:  "위조된 X-Forwarded-For 는 X-Real-IP 를 못 이긴다",
			rules: []string{"8.8.8.8"},
			headers: map[string]string{
				"X-Forwarded-For": "8.8.8.8",
				"X-Real-IP":       "1.1.1.1",
			},
			want: http.StatusOK,
		},
		{
			// 반대로 차단 대상이 XFF 를 심어 자기 자신을 숨기려 해도 소용없다.
			name:  "X-Real-IP 가 차단 대상이면 XFF 로 우회 불가",
			rules: []string{"8.8.8.8"},
			headers: map[string]string{
				"X-Forwarded-For": "1.1.1.1",
				"X-Real-IP":       "8.8.8.8",
			},
			want: http.StatusForbidden,
		},
		{
			name:    "X-Real-IP 가 없으면 XFF 첫 값으로 판정",
			rules:   []string{"8.8.8.8"},
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
			rules:   []string{"8.8.8.8"},
			headers: map[string]string{"X-Real-IP": "2001:db8::1"},
			want:    http.StatusOK,
		},
	}

	app := newGuardedApp()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seedBlocklist(t, tt.rules...)

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
