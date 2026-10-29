package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestReadonlyGuardMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name          string
		config        string
		adminOverride string
		wantStatus    int
		wantServed    bool
	}{
		{"readonly mode", "uiMode: readonly\n", "", http.StatusForbidden, false},
		{"readonly mode with admin override", "uiMode: readonly\n", "true", http.StatusOK, true},
		{"readonly mode with non-true override", "uiMode: readonly\n", "1", http.StatusForbidden, false},
		{"default mode", "uiMode: default\n", "", http.StatusOK, true},
		{"unset mode", "clusterName: prod\n", "", http.StatusOK, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cm := &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
				Data:       map[string]string{consts.OdigosConfigurationFileName: tc.config},
			}
			kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(cm)})
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
