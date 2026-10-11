package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// serveProtected mirrors how serveClientFiles in frontend/server/router.go runs the
// middleware by hand: it invokes it and then gates the protected handler on
// c.IsAborted(), so writing a response is not enough to stop the request.
func serveProtected(t *testing.T) (*httptest.ResponseRecorder, *bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	served := false
	r := gin.New()
	r.GET("/", func(c *gin.Context) {
		OidcMiddleware(context.Background())(c)
		if c.IsAborted() {
			return
		}
		served = true
		c.String(http.StatusOK, "spa bundle")
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	return w, &served
}

func TestOidcMiddleware_AbortsWhenConfigCannotBeRead(t *testing.T) {
	clientset := k8sfake.NewSimpleClientset()
	clientset.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("etcd leader changed")
	})
	kube.SetDefaultClient(&kube.Client{Interface: clientset})
	t.Cleanup(func() { kube.SetDefaultClient(nil) })

	w, served := serveProtected(t)

	if *served {
		t.Fatal("protected handler ran even though the oidc check could not complete")
	}
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestOidcMiddleware_ServesWhenOidcIsNotConfigured(t *testing.T) {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
		Data:       map[string]string{consts.OdigosConfigurationFileName: "clusterName: prod\n"},
	}
	kube.SetDefaultClient(&kube.Client{Interface: k8sfake.NewSimpleClientset(cm)})
	t.Cleanup(func() { kube.SetDefaultClient(nil) })

	w, served := serveProtected(t)

	if !*served {
		t.Fatal("protected handler did not run even though oidc is not configured")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
