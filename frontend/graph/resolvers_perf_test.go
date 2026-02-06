package graph

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const odigosNs = "odigos-system"

func TestMain(m *testing.M) {
	os.Setenv("CURRENT_NS", odigosNs)
	os.Exit(m.Run())
}

func slowReactor(latency time.Duration) k8stesting.ReactionFunc {
	return func(action k8stesting.Action) (bool, runtime.Object, error) {
		time.Sleep(latency)
		return false, nil, nil
	}
}

func odigosConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: odigosNs,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: "ignoredNamespaces: []",
		},
	}
}

func generateDestinationsAndSecrets(count int) ([]runtime.Object, []runtime.Object) {
	var odigosObjs []runtime.Object
	var k8sObjs []runtime.Object
	for i := 0; i < count; i++ {
		secretName := fmt.Sprintf("dest-secret-%d", i)
		destName := fmt.Sprintf("dest-%d", i)

		k8sObjs = append(k8sObjs, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: odigosNs,
			},
			Data: map[string][]byte{
				"api_key":  []byte(fmt.Sprintf("key-%d", i)),
				"endpoint": []byte(fmt.Sprintf("https://endpoint-%d.example.com", i)),
			},
		})

		odigosObjs = append(odigosObjs, &odigosv1alpha1.Destination{
			ObjectMeta: metav1.ObjectMeta{
				Name:      destName,
				Namespace: odigosNs,
			},
			Spec: odigosv1alpha1.DestinationSpec{
				Type:            "jaeger",
				DestinationName: fmt.Sprintf("Dest %d", i),
				Data:            map[string]string{"host": "localhost"},
				SecretRef:       &corev1.LocalObjectReference{Name: secretName},
			},
		})
	}
	return odigosObjs, k8sObjs
}

// BenchmarkDestinationSecrets_Before measures the old N+1 pattern:
// for each destination, call GetDestinationSecretFields (1 API call per dest).
func BenchmarkDestinationSecrets_Before(b *testing.B) {
	for _, destCount := range []int{10, 100} {
		b.Run(fmt.Sprintf("%ddests", destCount), func(b *testing.B) {
			latency := 5 * time.Millisecond
			odigosObjs, k8sObjs := generateDestinationsAndSecrets(destCount)
			k8sObjs = append(k8sObjs, odigosConfigMap())

			k8sFake := kubefake.NewSimpleClientset(k8sObjs...)
			k8sFake.PrependReactor("*", "*", slowReactor(latency))

			odigosFake := odigosfake.NewSimpleClientset(odigosObjs...)
			odigosFake.PrependReactor("*", "*", slowReactor(latency))

			kube.DefaultClient = &kube.Client{
				Interface:    k8sFake,
				OdigosClient: odigosFake.OdigosV1alpha1(),
			}

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Simulate old pattern: List destinations, then per-dest secret fetch
				dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}
				for _, dest := range dests.Items {
					_, err := services.GetDestinationSecretFields(ctx, odigosNs, &dest)
					if err != nil {
						b.Fatal(err)
					}
				}
				if len(dests.Items) != destCount {
					b.Fatalf("expected %d dests, got %d", destCount, len(dests.Items))
				}
			}
		})
	}
}

// BenchmarkDestinationSecrets_After measures the new batch pattern:
// one Secrets.List + one Destinations.List, then map lookups.
func BenchmarkDestinationSecrets_After(b *testing.B) {
	for _, destCount := range []int{10, 100} {
		b.Run(fmt.Sprintf("%ddests", destCount), func(b *testing.B) {
			latency := 5 * time.Millisecond
			odigosObjs, k8sObjs := generateDestinationsAndSecrets(destCount)
			k8sObjs = append(k8sObjs, odigosConfigMap())

			k8sFake := kubefake.NewSimpleClientset(k8sObjs...)
			k8sFake.PrependReactor("*", "*", slowReactor(latency))

			odigosFake := odigosfake.NewSimpleClientset(odigosObjs...)
			odigosFake.PrependReactor("*", "*", slowReactor(latency))

			kube.DefaultClient = &kube.Client{
				Interface:    k8sFake,
				OdigosClient: odigosFake.OdigosV1alpha1(),
			}

			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// New batch pattern: 1 Destinations.List + 1 Secrets.List
				dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}

				allSecrets, err := kube.DefaultClient.CoreV1().Secrets(odigosNs).List(ctx, metav1.ListOptions{})
				if err != nil {
					b.Fatal(err)
				}

				secretsByName := make(map[string]*corev1.Secret)
				for idx := range allSecrets.Items {
					secretsByName[allSecrets.Items[idx].Name] = &allSecrets.Items[idx]
				}

				for _, dest := range dests.Items {
					if dest.Spec.SecretRef != nil {
						if secret, ok := secretsByName[dest.Spec.SecretRef.Name]; ok {
							fields := services.ExtractSecretFields(secret)
							if len(fields) == 0 {
								b.Fatal("expected secret fields")
							}
						}
					}
				}
				if len(dests.Items) != destCount {
					b.Fatalf("expected %d dests, got %d", destCount, len(dests.Items))
				}
			}
		})
	}
}
