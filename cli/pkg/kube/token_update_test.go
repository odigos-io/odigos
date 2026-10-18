package kube

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func TestRemoteTokenUpdatePreservesKubernetesAuthentication(t *testing.T) {
	t.Setenv(consts.UIIDTokenEnvVarName, "review-id-token")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/namespaces/odigos-system/services/ui:3000/proxy/token/update" {
			t.Errorf("wrong proxy path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer kubernetes-token" {
			t.Error("Kubernetes authorization was replaced")
		}
		cookie, err := r.Cookie("id_token")
		if err != nil || cookie.Value != "review-id-token" {
			t.Error("UI identity was not forwarded")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	clientset, err := kubernetes.NewForConfig(&rest.Config{Host: server.URL, BearerToken: "kubernetes-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := ExecuteRemoteUpdateToken(context.Background(), &Client{Clientset: clientset}, "odigos-system", "license-token"); err != nil {
		t.Fatal(err)
	}
}
