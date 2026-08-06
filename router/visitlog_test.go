package router

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type visitRecorder struct {
	mutex sync.Mutex
	calls []string
}

func (r *visitRecorder) record(address string, path string, agent string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.calls = append(r.calls, address)
}

func (r *visitRecorder) count() int {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.calls)
}

// newVisitApp 은 DB 적재 대신 recorder 로 호출만 붙잡는 테스트 앱을 만든다.
func newVisitApp(t *testing.T, recorder *visitRecorder) *fiber.App {
	t.Helper()

	original := recordVisit
	recordVisit = recorder.record
	t.Cleanup(func() { recordVisit = original })

	app := fiber.New()
	app.Use(VisitLog)
	app.Get("/api/projects", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	return app
}

func visitRequest(t *testing.T, app *fiber.App, address string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	req.Header.Set("X-Real-IP", address)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
}

func TestVisitLogRecordsEveryRequest(t *testing.T) {
	recorder := &visitRecorder{}
	app := newVisitApp(t, recorder)

	visitRequest(t, app, "1.2.3.4")
	visitRequest(t, app, "1.2.3.4")
	visitRequest(t, app, "5.6.7.8")

	if got := recorder.count(); got != 3 {
		t.Fatalf("recorded %d visits, want 3 (요청마다 적재)", got)
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

	if got := recorder.count(); got != 0 {
		t.Fatalf("recorded %d visits, want 0 (OPTIONS 는 방문이 아니다)", got)
	}
}
