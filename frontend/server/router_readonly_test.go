package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/middlewares"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// Every non-GET REST route must refuse to run in read-only mode, and must keep
// reaching its handler otherwise.
func TestRouter_ReadonlyGuardsRestWrites(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const centralProxyToken = "central-proxy-token"
	// handlerRan reports whether the route's handler was reached: the token handler
	// rejects the malformed token, the source handlers call the Odigos API.
	routes := []struct {
		method, path, body string
		handlerRan         func(w *httptest.ResponseRecorder, odigosClient *odigosfake.Clientset) bool
	}{
		{http.MethodPost, "/token/update", `{"token":"not-a-token"}`, func(w *httptest.ResponseRecorder, _ *odigosfake.Clientset) bool {
			return w.Code == http.StatusInternalServerError && strings.Contains(w.Body.String(), "invalid JWT token format")
		}},
		{http.MethodPost, "/source/namespace/default/kind/deployment/name/app", "", func(_ *httptest.ResponseRecorder, odigosClient *odigosfake.Clientset) bool {
			return len(odigosClient.Actions()) > 0
		}},
		{http.MethodDelete, "/source/namespace/default/kind/deployment/name/app", "", func(_ *httptest.ResponseRecorder, odigosClient *odigosfake.Clientset) bool {
			return len(odigosClient.Actions()) > 0
		}},
	}
	modes := []struct {
		name          string
		config        string
		adminOverride bool
		proxyToken    string
		wantBlocked   bool
	}{
		{"readonly", "uiMode: readonly\n", false, "", true},
		{"readonly with admin override header only", "uiMode: readonly\n", true, "", true},
		{"readonly with admin override from central-proxy", "uiMode: readonly\n", true, centralProxyToken, false},
		{"default", "uiMode: default\n", false, "", false},
	}
	for _, mode := range modes {
		for _, route := range routes {
			t.Run(mode.name+" "+route.method+" "+route.path, func(t *testing.T) {
				clientset := k8sfake.NewSimpleClientset(&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
					Data:       map[string]string{consts.OdigosConfigurationFileName: mode.config},
				})
				clientset.PrependReactor("create", "tokenreviews", func(action k8stesting.Action) (bool, runtime.Object, error) {
					review := action.(k8stesting.CreateAction).GetObject().(*authenticationv1.TokenReview).DeepCopy()
					if review.Spec.Token == centralProxyToken {
						review.Status = authenticationv1.TokenReviewStatus{
							Authenticated: true,
							User:          authenticationv1.UserInfo{Username: "system:serviceaccount:" + env.GetCurrentNamespace() + ":central-proxy"},
							Audiences:     review.Spec.Audiences,
						}
					}
					return true, review, nil
				})
				odigosClient := odigosfake.NewSimpleClientset()
				previous := kube.DefaultClient
				kube.SetDefaultClient(&kube.Client{Interface: clientset, OdigosClient: odigosClient.OdigosV1alpha1()})
				t.Cleanup(func() { kube.SetDefaultClient(previous) })

				r, err := BuildRouter(t.Context(), &Deps{}, RouterOpts{})
				if err != nil {
					t.Fatal(err)
				}
				req := httptest.NewRequest(route.method, route.path, strings.NewReader(route.body))
				req.Header.Set("Content-Type", "application/json")
				if mode.adminOverride {
					req.Header.Set(middlewares.AdminOverrideHeader, "true")
				}
				if mode.proxyToken != "" {
					req.Header.Set(middlewares.ProxyTokenHeader, mode.proxyToken)
				}
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				blocked := w.Code == http.StatusForbidden && strings.Contains(w.Body.String(), services.ErrorIsReadonly.Error())
				if blocked != mode.wantBlocked {
					t.Fatalf("blocked = %v, want %v (status %d: %s)", blocked, mode.wantBlocked, w.Code, w.Body.String())
				}
				handlerRan := route.handlerRan(w, odigosClient)
				if handlerRan == mode.wantBlocked {
					t.Fatalf("handler ran = %v, want %v (status %d: %s)", handlerRan, !mode.wantBlocked, w.Code, w.Body.String())
				}
				for _, action := range clientset.Actions() {
					if mode.wantBlocked && action.GetVerb() != "get" {
						t.Fatalf("read-only request changed the cluster: %s %s", action.GetVerb(), action.GetResource().Resource)
					}
				}
			})
		}
	}
}
