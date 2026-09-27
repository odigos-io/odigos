package pro

import (
	"context"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestMarketplaceTokenUpdateDoesNotMutateSecret(t *testing.T) {
	client := fake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosProSecretName, Namespace: "odigos"},
		Data:       map[string][]byte{"aws-marketplace": []byte("true")},
	})
	err := updateSecretToken(context.Background(), client, "odigos", "replacement-token")
	if err == nil || !strings.Contains(err.Error(), "AWS Marketplace") {
		t.Fatalf("expected Marketplace activation error, got %v", err)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() != "get" {
			t.Fatalf("unexpected mutation: %v", action)
		}
	}
}
