package middlewares

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const validProxyToken = "valid-proxy-token"

// newFakeClientWithTokenReviews answers TokenReviews like an audience-aware API
// server where only validProxyToken is central-proxy's token for audience odigos-ui.
func newFakeClientWithTokenReviews(t *testing.T, config string, reviews *int) *k8sfake.Clientset {
	t.Helper()
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
		Data:       map[string]string{consts.OdigosConfigurationFileName: config},
	}
	client := k8sfake.NewSimpleClientset(cm)
	client.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
		*reviews++
		review := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenReview).DeepCopy()
		switch review.Spec.Token {
		case validProxyToken:
			review.Status = authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User:          authenticationv1.UserInfo{Username: "system:serviceaccount:" + env.GetCurrentNamespace() + ":central-proxy"},
				Audiences:     review.Spec.Audiences,
			}
		case "other-sa-token":
			review.Status = authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User:          authenticationv1.UserInfo{Username: "system:serviceaccount:" + env.GetCurrentNamespace() + ":odigos-ui"},
				Audiences:     review.Spec.Audiences,
			}
		case "other-namespace-token":
			review.Status = authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User:          authenticationv1.UserInfo{Username: "system:serviceaccount:other:central-proxy"},
				Audiences:     review.Spec.Audiences,
			}
		case "api-server-audience-token":
			review.Status = authenticationv1.TokenReviewStatus{
				Authenticated: true,
				User:          authenticationv1.UserInfo{Username: "system:serviceaccount:" + env.GetCurrentNamespace() + ":central-proxy"},
				Audiences:     []string{"https://kubernetes.default.svc"},
			}
		case "review-error-token":
			return true, nil, errors.New("tokenreviews.authentication.k8s.io is forbidden")
		default:
			review.Status = authenticationv1.TokenReviewStatus{Error: "invalid bearer token"}
		}
		return true, review, nil
	})
	return client
}

func resetProxyTokenCache(t *testing.T) {
	t.Helper()
	proxyTokenCacheMu.Lock()
	proxyTokenCache = map[[sha256.Size]byte]time.Time{}
	proxyTokenCacheMu.Unlock()
}

func TestReadonlyGuardMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name          string
		config        string
		adminOverride string
		proxyToken    string
		wantStatus    int
		wantServed    bool
	}{
		{"readonly mode", "uiMode: readonly\n", "", "", http.StatusForbidden, false},
		{"readonly mode with override header only", "uiMode: readonly\n", "true", "", http.StatusForbidden, false},
		{"readonly mode with override and central-proxy token", "uiMode: readonly\n", "true", validProxyToken, http.StatusOK, true},
		{"readonly mode with central-proxy token but no override header", "uiMode: readonly\n", "", validProxyToken, http.StatusForbidden, false},
		{"readonly mode with non-true override", "uiMode: readonly\n", "1", validProxyToken, http.StatusForbidden, false},
		{"readonly mode with invalid token", "uiMode: readonly\n", "true", "forged-token", http.StatusForbidden, false},
		{"readonly mode with another service account's token", "uiMode: readonly\n", "true", "other-sa-token", http.StatusForbidden, false},
		{"readonly mode with central-proxy token from another namespace", "uiMode: readonly\n", "true", "other-namespace-token", http.StatusForbidden, false},
		{"readonly mode with token for another audience", "uiMode: readonly\n", "true", "api-server-audience-token", http.StatusForbidden, false},
		{"readonly mode when the token review fails", "uiMode: readonly\n", "true", "review-error-token", http.StatusForbidden, false},
		{"default mode", "uiMode: default\n", "", "", http.StatusOK, true},
		{"unset mode", "clusterName: prod\n", "", "", http.StatusOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetProxyTokenCache(t)
			reviews := 0
			kube.SetDefaultClient(&kube.Client{Interface: newFakeClientWithTokenReviews(t, tc.config, &reviews)})
			t.Cleanup(func() { kube.SetDefaultClient(nil) })

			served := false
			r := gin.New()
			r.POST("/write", ReadonlyGuardMiddleware(), func(c *gin.Context) {
				served = true
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodPost, "/write", nil)
			if tc.adminOverride != "" {
				req.Header.Set(AdminOverrideHeader, tc.adminOverride)
			}
			if tc.proxyToken != "" {
				req.Header.Set(ProxyTokenHeader, tc.proxyToken)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tc.wantStatus)
			}
			if served != tc.wantServed {
				t.Fatalf("handler served = %v, want %v", served, tc.wantServed)
			}
			if !tc.wantServed && !strings.Contains(w.Body.String(), services.ErrorIsReadonly.Error()) {
				t.Fatalf("body %q does not carry the read-only error", w.Body.String())
			}
		})
	}
}

func TestWithAdminOverrideFromRequestCachesOnlySuccessfulReviews(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetProxyTokenCache(t)
	reviews := 0
	kube.SetDefaultClient(&kube.Client{Interface: newFakeClientWithTokenReviews(t, "uiMode: readonly\n", &reviews)})
	t.Cleanup(func() { kube.SetDefaultClient(nil) })

	override := func(token string) bool {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/graphql", nil)
		c.Request.Header.Set(AdminOverrideHeader, "true")
		c.Request.Header.Set(ProxyTokenHeader, token)
		WithAdminOverrideFromRequest(c)
		return AdminOverrideFromContext(c.Request.Context())
	}

	for i := 0; i < 2; i++ {
		if !override(validProxyToken) {
			t.Fatalf("request %d: central-proxy token did not grant the override", i)
		}
	}
	if reviews != 1 {
		t.Fatalf("token reviews = %d after two requests with the same valid token, want 1", reviews)
	}

	for i := 0; i < 2; i++ {
		if override("forged-token") {
			t.Fatalf("request %d: forged token granted the override", i)
		}
	}
	if reviews != 3 {
		t.Fatalf("token reviews = %d, want a fresh review for every rejected token (3)", reviews)
	}

	proxyTokenCacheMu.Lock()
	for key, expiry := range proxyTokenCache {
		proxyTokenCache[key] = expiry.Add(-2 * proxyTokenCacheTTL)
	}
	proxyTokenCacheMu.Unlock()
	if !override(validProxyToken) || reviews != 4 {
		t.Fatalf("expired cache entry was not re-reviewed (reviews = %d, want 4)", reviews)
	}
}
