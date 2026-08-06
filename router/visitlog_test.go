package router

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gowoobro/models"

	"github.com/gofiber/fiber/v2"
)

type visitRecorder struct {
	mutex    sync.Mutex
	records  []models.Iplog
	notifies []models.Iplog
}

func (r *visitRecorder) record(item models.Iplog) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.records = append(r.records, item)
}

func (r *visitRecorder) notify(item models.Iplog) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.notifies = append(r.notifies, item)
}

func (r *visitRecorder) counts() (int, int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.records), len(r.notifies)
}

// newVisitApp 은 DB 적재·ntfy 발송 대신 recorder 로 호출만 붙잡는 테스트 앱을
// 만든다. 알림 dedupe 캐시도 비워서 테스트끼리 간섭하지 않게 한다.
func newVisitApp(t *testing.T, recorder *visitRecorder) *fiber.App {
	t.Helper()

	notified.mutex.Lock()
	notified.seen = map[string]time.Time{}
	notified.mutex.Unlock()

	originalRecord := recordVisit
	originalNotify := notifyVisit
	recordVisit = recorder.record
	notifyVisit = recorder.notify
	t.Cleanup(func() {
		recordVisit = originalRecord
		notifyVisit = originalNotify
	})

	app := fiber.New()
	app.Use(VisitLog)
	app.Get("/api/projects", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	return app
}

func visitRequest(t *testing.T, app *fiber.App, address string, agent string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("X-Real-IP", address)
	if agent != "" {
		req.Header.Set("User-Agent", agent)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

const chromeAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"
const googlebotAgent = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"

func TestVisitLogRecordsEveryRequest(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)

	visitRequest(t, app, "1.2.3.4", chromeAgent)
	visitRequest(t, app, "1.2.3.4", chromeAgent)
	visitRequest(t, app, "5.6.7.8", chromeAgent)

	records, _ := recorder.counts()
	if records != 3 {
		t.Fatalf("recorded %d visits, want 3 (적재는 요청마다)", records)
	}
}

func TestVisitLogNotifiesOncePerIP(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)

	visitRequest(t, app, "1.2.3.4", chromeAgent)
	visitRequest(t, app, "1.2.3.4", chromeAgent)
	visitRequest(t, app, "5.6.7.8", chromeAgent)

	_, notifies := recorder.counts()
	if notifies != 2 {
		t.Fatalf("notified %d times, want 2 (알림은 TTL 안의 같은 IP 는 한 번만)", notifies)
	}
}

func TestVisitLogNotifiesAgainAfterTTL(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)

	visitRequest(t, app, "1.2.3.4", chromeAgent)

	// TTL 이 지난 것처럼 마지막 알림 시각을 과거로 되돌린다.
	notified.mutex.Lock()
	notified.seen["1.2.3.4"] = time.Now().Add(-notifyTTL - time.Minute)
	notified.mutex.Unlock()

	visitRequest(t, app, "1.2.3.4", chromeAgent)

	_, notifies := recorder.counts()
	if notifies != 2 {
		t.Fatalf("notified %d times, want 2 (TTL 이 지나면 다시 알림)", notifies)
	}
}

func TestVisitLogSkipsBotNotification(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)

	visitRequest(t, app, "66.249.73.12", googlebotAgent)

	records, notifies := recorder.counts()
	if records != 1 {
		t.Fatalf("recorded %d visits, want 1 (봇도 적재는 한다)", records)
	}
	if notifies != 0 {
		t.Fatalf("notified %d times, want 0 (봇은 알리지 않는다)", notifies)
	}
}

func TestParseAgent(t *testing.T) {
	cases := []struct {
		name    string
		agent   string
		os      string
		browser string
		bot     bool
	}{
		{
			name:    "윈도우 크롬",
			agent:   chromeAgent,
			os:      "Windows 10.0",
			browser: "Chrome 126",
			bot:     false,
		},
		{
			name:    "아이폰 사파리",
			agent:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			os:      "iOS 17.5",
			browser: "Safari 17",
			bot:     false,
		},
		{
			name:    "구글봇",
			agent:   googlebotAgent,
			os:      "",
			browser: "Googlebot 2",
			bot:     true,
		},
		{
			name:    "curl",
			agent:   "curl/8.7.1",
			os:      "",
			browser: "curl 8",
			bot:     false,
		},
		{
			name:    "빈 값",
			agent:   "",
			os:      "",
			browser: "",
			bot:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			os, browser, bot := parseAgent(c.agent)
			if os != c.os || browser != c.browser || bot != c.bot {
				t.Fatalf("parseAgent() = (%q, %q, %v), want (%q, %q, %v)", os, browser, bot, c.os, c.browser, c.bot)
			}
		})
	}
}

func TestVisitLogSkipsPreflight(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)
	app.Options("/api/projects", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/projects", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	if _, err := app.Test(req); err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}

	records, notifies := recorder.counts()
	if records != 0 || notifies != 0 {
		t.Fatalf("records=%d notifies=%d, want 0/0 (OPTIONS 는 방문이 아니다)", records, notifies)
	}
}
