package remote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
)

func TestRemoteUIAuthentication(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusServiceUnavailable} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Setenv(consts.UIIDTokenEnvVarName, "review-id-token")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer review-id-token" {
					t.Errorf("missing bearer identity")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"numberOfDestinations":1}`))
			}))
			defer server.Close()
			u, _ := url.Parse(server.URL)
			previousPort := DefaultLocalPort
			DefaultLocalPort = u.Port()
			t.Cleanup(func() { DefaultLocalPort = previousPort })
			for _, call := range []struct {
				name string
				run  func() error
			}{
				{"destination count", func() error { _, err := GetNumberOfDestinations(context.Background()); return err }},
				{"describe", func() error { _, err := DescribeOdigos(context.Background()); return err }},
				{"describe source", func() error {
					_, err := DescribeSource(context.Background(), nil, "odigos-system", "Deployment", "default", "app")
					return err
				}},
				{"create source", func() error {
					return CreateSource(context.Background(), nil, "odigos-system", "Deployment", "default", "app")
				}},
			} {
				t.Run(call.name, func(t *testing.T) {
					err := call.run()
					if (err == nil) != (status == http.StatusOK) {
						t.Fatalf("unexpected error: %v", err)
					}
					if err != nil && strings.Contains(err.Error(), "review-id-token") {
						t.Fatal("error exposed ID token")
					}
				})
			}
		})
	}
}
func TestRemoteUIWithoutOidcToken(t *testing.T) {
	t.Setenv(consts.UIIDTokenEnvVarName, "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("unexpected authorization")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	resp, err := doUIRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
}
