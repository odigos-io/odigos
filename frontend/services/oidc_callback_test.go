package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestOidcAuthCallback_ProviderDiscoveryFailure(t *testing.T) {
	setOidcTestClient(t, []runtime.Object{effectiveConfigCM(fullOidcConfig), oidcSecret("secret")})
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/auth/oidc-callback", func(c *gin.Context) { OidcAuthCallback(context.Background(), c) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/oidc-callback", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status %d", w.Code)
	}
}

func TestGetOidcOauthConfig_InvalidConfigurationFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name, config string
		secret       string
	}{
		{"deleted secret", fullOidcConfig, ""},
		{"empty secret", fullOidcConfig, " "},
		{"missing client ID", "oidc:\n  tenantUrl: https://idp.example", "secret"},
		{"missing tenant", "oidc:\n  clientId: ui", "secret"},
		{"missing config key", "", "secret"},
		{"empty config YAML", " ", "secret"},
		{"empty oidc block", "oidc: {}", "secret"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cm := effectiveConfigCM(tc.config)
			if tc.name == "missing config key" {
				cm.Data = nil
			}
			objs := []runtime.Object{cm}
			if tc.secret != "" {
				objs = append(objs, oidcSecret(tc.secret))
			}
			setOidcTestClient(t, objs)
			cfg, err := GetOidcOauthConfig(context.Background())
			if cfg != nil || err == nil {
				t.Fatalf("expected configuration error, got %v %v", cfg, err)
			}
		})
	}
}

func TestGetOidcOauthConfig_CanceledMissingConfig(t *testing.T) {
	setOidcTestClient(t, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetOidcOauthConfig(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestOidcRequestContext_ClientCancellation(t *testing.T) {
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	ctx, cancel := OidcRequestContext(context.Background(), requestCtx)
	defer cancel()
	cancelRequest()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("request cancellation did not stop authentication")
	}
}

// go-oidc uses context.WithoutCancel for JWKS fetches; they must still stop
// through the configured HTTP transport when the originating request ends.
func TestOidcRequestContext_DetachedKeyFetchCancellation(t *testing.T) {
	started := make(chan struct{})
	requestCtx, cancelRequest := context.WithCancel(context.Background())
	defer cancelRequest()
	serverCtx := oidc.ClientContext(context.Background(), &http.Client{Transport: oidcBlockingTestTransport{started: started}})
	ctx, cancel := OidcRequestContext(serverCtx, requestCtx)
	defer cancel()
	detached := context.WithoutCancel(ctx)
	client := detached.Value(oauth2.HTTPClient).(*http.Client)
	result := make(chan error, 1)
	go func() {
		req, _ := http.NewRequestWithContext(detached, http.MethodGet, "https://idp.example/keys", nil)
		_, err := client.Do(req)
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("key fetch did not start")
	}
	cancelRequest()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled key fetch, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("key fetch outlived its request")
	}
}

type oidcBlockingTestTransport struct{ started chan struct{} }

func (t oidcBlockingTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	close(t.started)
	<-req.Context().Done()
	return nil, req.Context().Err()
}
