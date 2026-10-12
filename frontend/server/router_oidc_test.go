package server

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func TestOidcRouter_Authentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := map[string]any{"keys": []any{map[string]any{
		"kty": "RSA", "kid": "review", "alg": "RS256", "use": "sig",
		"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB",
	}}}
	var issuer, validToken string
	idp := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "token_type": "Bearer", "id_token": validToken})
			return
		}
		if r.URL.Path == "/keys" {
			_ = json.NewEncoder(w).Encode(jwks)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": issuer, "authorization_endpoint": issuer + "/authorize",
			"token_endpoint": issuer + "/token", "jwks_uri": issuer + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	}))
	defer idp.Close()
	issuer = idp.URL
	validToken = signOidcTestToken(t, key, map[string]any{
		"iss": issuer, "aud": "ui", "sub": "review-user",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": "review-nonce",
	})
	expiredToken := signOidcTestToken(t, key, map[string]any{
		"iss": issuer, "aud": "ui", "sub": "review-user", "exp": time.Now().Add(-time.Hour).Unix(),
	})
	wrongAudience := signOidcTestToken(t, key, map[string]any{
		"iss": issuer, "aud": "other-client", "sub": "review-user", "exp": time.Now().Add(time.Hour).Unix(),
	})
	wrongIssuer := signOidcTestToken(t, key, map[string]any{
		"iss": "https://other-idp.example", "aud": "ui", "sub": "review-user", "exp": time.Now().Add(time.Hour).Unix(),
	})
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrongSignature := signOidcTestToken(t, otherKey, map[string]any{
		"iss": issuer, "aud": "ui", "sub": "review-user", "exp": time.Now().Add(time.Hour).Unix(),
	})
	ctx := oidc.ClientContext(context.Background(), idp.Client())
	fullConfig := fmt.Sprintf("oidc:\n  tenantUrl: %s\n  clientId: ui\n  clientSecret: secretRef:odigos-oidc\n", issuer)
	cases := []struct {
		name, config, token, authorization, method, path string
		secret                                           bool
		status                                           int
		served                                           bool
	}{
		{"no OIDC SPA", "clusterName: prod", "", "", "GET", "/", false, 200, true},
		{"no OIDC API", "clusterName: prod", "", "", "POST", "/graphql", false, 200, false},
		{"login redirect", fullConfig, "", "", "GET", "/", true, 302, false},
		{"static asset redirect", fullConfig, "", "", "GET", "/_next/main.js", true, 302, false},
		{"invalid token", fullConfig, "forged", "", "GET", "/", true, 401, false},
		{"wrong audience", fullConfig, wrongAudience, "", "GET", "/", true, 401, false},
		{"wrong issuer", fullConfig, wrongIssuer, "", "GET", "/", true, 401, false},
		{"wrong signature", fullConfig, wrongSignature, "", "GET", "/", true, 401, false},
		{"expired cookie", fullConfig, expiredToken, "", "GET", "/", true, 401, false},
		{"valid callback", fullConfig, "", "", "GET", "/auth/oidc-callback?code=review-code&state=review-state", true, 302, false},
		{"valid cookie SPA", fullConfig, validToken, "", "GET", "/", true, 200, true},
		{"valid cookie API", fullConfig, validToken, "", "POST", "/graphql", true, 200, false},
		{"valid bearer API", fullConfig, "", "Bearer " + validToken, "POST", "/graphql", true, 200, false},
		{"malformed bearer API", fullConfig, validToken, "Basic abc", "POST", "/graphql", true, 401, false},
		{"invalid bearer API", fullConfig, "", "Bearer forged", "POST", "/graphql", true, 401, false},
		{"identity in extra mount", fullConfig, validToken, "", "GET", "/extra-auth", true, 200, false},
		{"deleted secret SPA", fullConfig, "", "", "GET", "/", false, 503, false},
		{"deleted secret API", fullConfig, "", "", "POST", "/graphql", false, 503, false},
		{"public health with broken OIDC", fullConfig, "", "", "GET", "/healthz", false, 200, false},
		{"public ready with broken OIDC", fullConfig, "", "", "GET", "/readyz", false, 200, false},
		{"public CSRF with broken OIDC", fullConfig, "", "", "GET", "/auth/csrf-token", false, 200, false},
		{"callback failure", fullConfig, "", "", "GET", "/auth/oidc-callback", false, 503, false},
	}
	for _, route := range []struct{ method, path string }{
		{"POST", "/graphql"}, {"GET", "/api/events"}, {"POST", "/token/update"},
		{"GET", "/workload"}, {"POST", "/source/namespace/default/kind/Deployment/name/app"},
		{"DELETE", "/source/namespace/default/kind/Deployment/name/app"},
		{"GET", "/describe/odigos"}, {"GET", "/playground"}, {"GET", "/diagnose/download"},
		{"GET", "/extra-auth"},
	} {
		cases = append(cases, struct {
			name, config, token, authorization, method, path string
			secret                                           bool
			status                                           int
			served                                           bool
		}{"unauthenticated " + route.method + " " + route.path, fullConfig, "", "", route.method, route.path, true, 401, false})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setOidcRouterClient(t, tc.config, tc.secret)
			r, err := BuildRouter(ctx, &Deps{}, RouterOpts{ExtraMounts: []func(*gin.Engine, *Deps){func(r *gin.Engine, _ *Deps) {
				r.GET("/extra-auth", func(c *gin.Context) {
					if c.GetString("uid") != "review-user" {
						t.Error("missing authenticated subject")
					}
					c.Status(200)
				})
			}}})
			if err != nil {
				t.Fatal(err)
			}
			serveClientFiles(r, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("SPA_MARKER")}, "_next/main.js": &fstest.MapFile{Data: []byte("SPA_MARKER")}})
			var body io.Reader
			if tc.path == "/graphql" {
				body = strings.NewReader(`{"query":"{__typename}"}`)
			}
			req := httptest.NewRequest(tc.method, tc.path, body)
			req.Header.Set("Content-Type", "application/json")
			if strings.Contains(tc.path, "code=review-code") {
				req.AddCookie(&http.Cookie{Name: "state", Value: "review-state"})
				req.AddCookie(&http.Cookie{Name: "nonce", Value: "review-nonce"})
			}
			if tc.token != "" {
				req.AddCookie(&http.Cookie{Name: "id_token", Value: tc.token})
			}
			if tc.authorization != "" {
				req.Header.Set("Authorization", tc.authorization)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
			if strings.Contains(w.Body.String(), "SPA_MARKER") != tc.served {
				t.Fatalf("unexpected SPA content: %s", w.Body.String())
			}
			if strings.Contains(tc.path, "code=review-code") {
				if w.Header().Get("Location") != "/" {
					t.Fatalf("wrong callback redirect: %s", w.Header().Get("Location"))
				}
				var session *http.Cookie
				for _, cookie := range w.Result().Cookies() {
					if cookie.Name == "id_token" {
						session = cookie
					}
				}
				if session == nil || session.Value != validToken || !session.HttpOnly {
					t.Fatal("callback did not set the session cookie")
				}
			} else if tc.status == 302 && !strings.HasPrefix(w.Header().Get("Location"), issuer+"/authorize") {
				t.Fatalf("wrong login redirect: %s", w.Header().Get("Location"))
			}
		})
	}
}

// OidcMiddleware runs on every application route and reads the effective-config
// ConfigMap to decide whether authentication is on. A cluster that never configured
// OIDC must therefore keep serving when that read fails transiently: GraphQL and SSE
// answer from the informer cache and do not need the API server at all, so rejecting
// them turns an API server blip into a cluster-wide UI outage.
func TestOidcRouter_TransientApiErrorKeepsServingUnconfiguredCluster(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ns := env.GetCurrentNamespace()
	clientset := k8sfake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: ns},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "clusterName: prod\n"},
	})
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: clientset})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	build := func() *gin.Engine {
		r, err := BuildRouter(context.Background(), &Deps{}, RouterOpts{})
		if err != nil {
			t.Fatal(err)
		}
		serveClientFiles(r, fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("SPA_MARKER")}})
		return r
	}
	call := func(r *gin.Engine, method, path string) int {
		var body io.Reader
		if path == "/graphql" {
			body = strings.NewReader(`{"query":"{__typename}"}`)
		}
		req := httptest.NewRequest(method, path, body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	// one healthy request, so the settings have been read at least once.
	if code := call(build(), "GET", "/"); code != http.StatusOK {
		t.Fatalf("expected the SPA to be served while the API server is healthy, got %d", code)
	}

	clientset.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("etcd leader changed")
	})

	r := build()
	for _, route := range []struct{ method, path string }{
		{"GET", "/"}, {"POST", "/graphql"}, {"GET", "/api/events"},
	} {
		if code := call(r, route.method, route.path); code == http.StatusServiceUnavailable {
			t.Errorf("%s %s rejected with 503 while OIDC is not configured", route.method, route.path)
		}
	}
}

func setOidcRouterClient(t *testing.T, config string, secret bool) {
	t.Helper()
	ns := env.GetCurrentNamespace()
	objs := []runtime.Object{&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: ns}, Data: map[string]string{consts.OdigosConfigurationFileName: config}}}
	if secret {
		objs = append(objs, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: consts.OidcSecretName, Namespace: ns}, Data: map[string][]byte{consts.OidcClientSecretProperty: []byte("secret")}})
	}
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(objs...)})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })
}

func signOidcTestToken(t *testing.T, key *rsa.PrivateKey, claims map[string]any) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","kid":"review"}`))
	body, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	unsigned := header + "." + base64.RawURLEncoding.EncodeToString(body)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}
