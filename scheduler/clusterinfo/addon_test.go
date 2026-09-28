package clusterinfo

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func TestInitializeAddonState(t *testing.T) {
	for _, populated := range []bool{false, true} {
		deployment := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
			Name: k8sconsts.OdigosDeploymentConfigMapName, Namespace: "odigos", UID: "installation-uid",
		}, Data: map[string]string{"ODIGOS_VERSION": "v1.0.0"}}
		offsets := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.GoOffsetsConfigMap, Namespace: "odigos"}}
		if populated {
			deployment.Data[k8sconsts.OdigosDeploymentConfigMapOdigosDeploymentIDKey] = "existing-installation"
			offsets.Data = map[string]string{k8sconsts.GoOffsetsFileName: "customer-offsets"}
		}
		client := fake.NewClientset(deployment, offsets)
		for range 2 {
			if err := InitializeAddonState(context.Background(), client, "odigos"); err != nil {
				t.Fatal(err)
			}
		}
		gotDeployment, _ := client.CoreV1().ConfigMaps("odigos").Get(context.Background(), deployment.Name, metav1.GetOptions{})
		gotOffsets, _ := client.CoreV1().ConfigMaps("odigos").Get(context.Background(), offsets.Name, metav1.GetOptions{})
		wantID, wantOffsets := "installation-uid", ""
		if populated {
			wantID, wantOffsets = "existing-installation", "customer-offsets"
		}
		if gotDeployment.Data[k8sconsts.OdigosDeploymentConfigMapOdigosDeploymentIDKey] != wantID || gotDeployment.Data["ODIGOS_VERSION"] != "v1.0.0" {
			t.Fatal("installation identity or release metadata changed")
		}
		if value, exists := gotOffsets.Data[k8sconsts.GoOffsetsFileName]; !exists || value != wantOffsets {
			t.Fatal("offsets were overwritten or not initialized")
		}
		updates := 0
		for _, action := range client.Actions() {
			if action.GetVerb() == "update" {
				updates++
			}
		}
		if (populated && updates != 0) || (!populated && updates != 2) {
			t.Fatalf("initialization is not idempotent: %d writes", updates)
		}
	}
}

func TestInitializeAddonStatePreservesConcurrentUpdate(t *testing.T) {
	deployment := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosDeploymentConfigMapName, Namespace: "odigos", UID: "uid"}}
	offsets := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.GoOffsetsConfigMap, Namespace: "odigos"}}
	client := fake.NewClientset(deployment, offsets)
	conflicted := false
	client.PrependReactor("update", "configmaps", func(action ktesting.Action) (bool, runtime.Object, error) {
		cm := action.(ktesting.UpdateAction).GetObject().(*corev1.ConfigMap)
		if cm.Name != offsets.Name || conflicted {
			return false, nil, nil
		}
		conflicted = true
		concurrent := offsets.DeepCopy()
		concurrent.Data = map[string]string{k8sconsts.GoOffsetsFileName: "concurrent-offsets"}
		if err := client.Tracker().Update(corev1.SchemeGroupVersion.WithResource("configmaps"), concurrent, "odigos"); err != nil {
			t.Fatal(err)
		}
		return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, offsets.Name, nil)
	})
	if err := InitializeAddonState(context.Background(), client, "odigos"); err != nil {
		t.Fatal(err)
	}
	got, _ := client.CoreV1().ConfigMaps("odigos").Get(context.Background(), offsets.Name, metav1.GetOptions{})
	if got.Data[k8sconsts.GoOffsetsFileName] != "concurrent-offsets" {
		t.Fatal("lost concurrent offsets")
	}
}

func TestInitializeAddonStateRequiresExistingResources(t *testing.T) {
	if err := InitializeAddonState(context.Background(), fake.NewClientset(), "odigos"); err == nil {
		t.Fatal("missing add-on resources were accepted")
	}
}
