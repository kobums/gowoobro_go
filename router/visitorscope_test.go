package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gowoobro/controllers"

	"github.com/gofiber/fiber/v2"
)

// newScopeApp 은 VisitorScope 가 심은 값을 그대로 응답으로 돌려준다.
func newScopeApp() *fiber.App {
	app := fiber.New()

	api := app.Group("/api")
	api.Use(VisitorScope)
	api.Get("/answers", func(c *fiber.Ctx) error {
		var ctl controllers.Controller
		ctl.Init(c)
		return c.SendString(ctl.VisitorAddress())
	})

	return app
}

func scopeBody(t *testing.T, app *fiber.App, req *http.Request) string {
	t.Helper()

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	return string(body)
}

func TestVisitorScopeUsesServerSideIP(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newScopeApp()

	// 방문자가 남의 IP 를 address 로 적어 보내도 서버가 본 IP 가 쓰인다.
	req := httptest.NewRequest(http.MethodGet, "/api/answers?address=9.9.9.9", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")

	if got := scopeBody(t, app, req); got != "1.2.3.4" {
		t.Fatalf("visitor address = %q, want 1.2.3.4", got)
	}
}

func TestVisitorScopeSkipsAdmin(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newScopeApp()

	token, err := issueToken()
	if err != nil {
		t.Fatalf("issueToken: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/answers", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.Header.Set("Authorization", "Bearer "+token)

	if got := scopeBody(t, app, req); got != "" {
		t.Fatalf("admin 요청에 visitor address %q 가 심겼다", got)
	}
}

func TestVisitorScopeIgnoresBadToken(t *testing.T) {
	withAuthConfig(t, "test-secret", "test-password")
	app := newScopeApp()

	req := httptest.NewRequest(http.MethodGet, "/api/answers", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")
	req.Header.Set("Authorization", "Bearer not-a-token")

	if got := scopeBody(t, app, req); got != "1.2.3.4" {
		t.Fatalf("visitor address = %q, want 1.2.3.4", got)
	}
}
