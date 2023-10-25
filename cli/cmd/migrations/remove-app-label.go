package migrations

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type removeAppLabelPatcher struct {
	Patcher

	resourceInterface  dynamic.ResourceInterface
	objectName         string
	expectedLabelValue string
}

func NewRemoveAppLabelPatcherDeployment(client *kube.Client, ns string, objectName string, expectedLabelValue string) Patcher {
	return &removeAppLabelPatcher{
		resourceInterface:  getResourceInterfaceDeployment(client, ns),
		objectName:         objectName,
		expectedLabelValue: expectedLabelValue,
	}
}

func NewRemoveAppLabelPatcherDaemonSet(client *kube.Client, ns string, objectName string, expectedLabelValue string) Patcher {
	return &removeAppLabelPatcher{
		resourceInterface:  getResourceInterfaceDaemonSet(client, ns),
		objectName:         objectName,
		expectedLabelValue: expectedLabelValue,
	}
}

func (p *removeAppLabelPatcher) PatcherName() string {
	return "RemoveAppLabel"
}

func (p *removeAppLabelPatcher) Patch(ctx context.Context) error {
	patchType, patchData := getPatchRemoveAppLabel(p.expectedLabelValue)
	_, err := p.resourceInterface.Patch(ctx, p.objectName, patchType, patchData, metav1.PatchOptions{})
	return err
}

func (p *removeAppLabelPatcher) UnPatch(ctx context.Context) error {
	patchType, patchData := getUnPatchRemoveAppLabel(p.expectedLabelValue)
	_, err := p.resourceInterface.Patch(ctx, p.objectName, patchType, patchData, metav1.PatchOptions{})
	return err
}

// the app label makes sense on pods to group them into a replicaset,
// but not on deployments or daemonsets where it doesn't mean anything.
// this patch removes the "app" label from those resources
func getPatchRemoveAppLabel(expectedValue string) (k8stypes.PatchType, []byte) {
	jsonBytes := encodeJsonPatchDocument(jsonPatchDocument{
		{
			Op:    "test",
			Path:  "/metadata/labels/app",
			Value: expectedValue,
		},
		{
			Op:   "remove",
			Path: "/metadata/labels/app",
		},
	})
	return k8stypes.JSONPatchType, jsonBytes
}

func getUnPatchRemoveAppLabel(expectedValue string) (k8stypes.PatchType, []byte) {
	jsonBytes := encodeJsonPatchDocument(jsonPatchDocument{
		{
			Op:    "add",
			Path:  "/metadata/labels/app",
			Value: expectedValue,
		},
	})
	return k8stypes.JSONPatchType, jsonBytes
}
