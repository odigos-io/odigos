package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestBuildRouterExtraMountsBeforeNoRoute asserts the load-bearing contract of
// the extension seam: an ExtraMounts handler is reachable (i.e. registered
// before the SPA NoRoute fallback), and the OSS health route is present. It uses
// a minimal Deps (no kube client) so the readiness check returns 503 — that is
// expected and still proves the route is wired.
func TestBuildRouterExtraMountsBeforeNoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mounted := false
	deps := &Deps{Flags: Flags{}}
	r, err := BuildRouter(t.Context(), deps, RouterOpts{
		ExtraMounts: []func(*gin.Engine, *Deps){
			func(e *gin.Engine, d *Deps) {
				mounted = true
				e.GET("/ext/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildRouter: %v", err)
	}
	if !mounted {
		t.Fatal("ExtraMounts callback was not invoked")
	}

	// The extra route resolves (would 404 via NoRoute if mounted after the SPA fallback).
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ext/ping", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Fatalf("extra mount not reachable before NoRoute: code=%d body=%q", rec.Code, rec.Body.String())
	}

	// A core OSS route is registered.
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code == http.StatusNotFound {
		t.Fatalf("/healthz not registered (got 404)")
	}
}

// TestBuildRouterNilMountIgnored ensures a nil entry in ExtraMounts is skipped.
func TestBuildRouterNilMountIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_, err := BuildRouter(t.Context(), &Deps{Flags: Flags{}}, RouterOpts{
		ExtraMounts: []func(*gin.Engine, *Deps){nil},
	})
	if err != nil {
		t.Fatalf("BuildRouter with nil mount: %v", err)
	}
}
