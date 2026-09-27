package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/middlewares"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func licenseTestClient(t *testing.T) *fake.Clientset {
	t.Helper()
	client := fake.NewClientset(
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()},
			Data: map[string]string{consts.OdigosConfigurationFileName: "configVersion: 1\nuiMode: default\n"}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosDeploymentConfigMapName, Namespace: env.GetCurrentNamespace()},
			Data: map[string]string{k8sconsts.OdigosDeploymentConfigMapInstallationStatusKey: "FINISHED"}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosProSecretName, Namespace: env.GetCurrentNamespace()},
			Data: map[string][]byte{k8sconsts.OdigosOnpremTokenSecretKey: []byte("legacy-token")}},
	)
	original := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{Interface: client})
	t.Cleanup(func() { kube.SetDefaultClient(original) })
	return client
}

func executeLicenseQuery(t *testing.T, query string) map[string]interface{} {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request = request.WithContext(middlewares.WithAdminOverride(request.Context()))
	response := httptest.NewRecorder()
	GetGQLHandler(context.Background(), NewExecutableSchema(Config{Resolvers: &Resolver{}})).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GraphQL status %d: %s", response.Code, response.Body)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestMarketplaceTokenQuerySkipsJWTAndSecretReads(t *testing.T) {
	t.Setenv("ODIGOS_LICENSE_PROVIDER", "aws-marketplace")
	client := licenseTestClient(t)
	result := executeLicenseQuery(t, `query {
      config { licenseProvider }
      platform: computePlatform { ...Tokens }
    }
    fragment Tokens on ComputePlatform { licenses: apiTokens { name token expiresAt message } }`)
	if errors := result["errors"]; errors != nil {
		t.Fatalf("unexpected errors: %v", errors)
	}
	data := result["data"].(map[string]interface{})
	if data["config"].(map[string]interface{})["licenseProvider"] != "aws-marketplace" {
		t.Fatal("provider missing")
	}
	if tokens := data["platform"].(map[string]interface{})["licenses"].([]interface{}); len(tokens) != 0 {
		t.Fatal("Marketplace exposed a token")
	}
	for _, action := range client.Actions() {
		if action.GetResource().Resource == "secrets" {
			t.Fatal("Marketplace token query read a secret")
		}
	}
}

func TestMarketplaceRejectsTokenMutationBeforeResolver(t *testing.T) {
	t.Setenv("ODIGOS_LICENSE_PROVIDER", "aws-marketplace")
	client := licenseTestClient(t)
	result := executeLicenseQuery(t, `mutation { replaceLicense: updateApiToken(token: "not-a-token") }`)
	errors, ok := result["errors"].([]interface{})
	if !ok || len(errors) != 1 || !strings.Contains(errors[0].(map[string]interface{})["message"].(string), "AWS Marketplace") {
		t.Fatalf("expected Marketplace error, got %v", result)
	}
	for _, action := range client.Actions() {
		if action.GetResource().Resource == "secrets" || action.GetVerb() != "get" {
			t.Fatalf("token mutation reached Kubernetes: %v", action)
		}
	}
}

func TestOrdinaryTokenQueryKeepsExistingBehavior(t *testing.T) {
	t.Setenv("ODIGOS_LICENSE_PROVIDER", "")
	client := licenseTestClient(t)
	result := executeLicenseQuery(t, `query { config { licenseProvider } computePlatform { apiTokens { token message } } }`)
	if errors := result["errors"]; errors != nil {
		t.Fatalf("unexpected errors: %v", errors)
	}
	data := result["data"].(map[string]interface{})
	if data["config"].(map[string]interface{})["licenseProvider"] != "odigos" {
		t.Fatal("legacy provider changed")
	}
	tokens := data["computePlatform"].(map[string]interface{})["apiTokens"].([]interface{})
	if len(tokens) != 1 || tokens[0].(map[string]interface{})["token"] != "legacy-token" {
		t.Fatal("legacy token path changed")
	}
	readSecret := false
	for _, action := range client.Actions() {
		readSecret = readSecret || action.GetResource().Resource == "secrets"
	}
	if !readSecret {
		t.Fatal("ordinary token resolver was skipped")
	}
}
