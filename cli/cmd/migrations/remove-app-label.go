package migrations

import (
	"context"

	"github.com/keyval-dev/odigos/cli/cmd/resources"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

type removeAppLabelPatcher struct {
}

func NewRemoveAppLabelMigrationStep() MigrationStep {
	return &removeAppLabelPatcher{}
}

func (p *removeAppLabelPatcher) SourceVersion() string {
	return "v0.1.81"
}

func (p *removeAppLabelPatcher) MigrationName() string {
	return "RemoveAppLabel"
}

// Migrate implements MigrationStep.
func (*removeAppLabelPatcher) Migrate(ctx context.Context, client *kube.Client, odigosNs string) error {

	patchType, patchDataAutoScaler := getPatchRemoveAppLabel(resources.AutoScalerAppLabelValue)
	_, err := client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.AutoScalerDeploymentName, patchType, patchDataAutoScaler, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchInstrumentor := getPatchRemoveAppLabel(resources.InstrumentorAppLabelValue)
	_, err = client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.InstrumentorDeploymentName, patchType, patchInstrumentor, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchSchedular := getPatchRemoveAppLabel(resources.SchedulerAppLabelValue)
	_, err = client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.SchedulerDeploymentName, patchType, patchSchedular, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchDataOdiglet := getPatchRemoveAppLabel(resources.OdigletAppLabelValue)
	_, err = client.AppsV1().DaemonSets(odigosNs).Patch(ctx, resources.OdigletDaemonSetName, patchType, patchDataOdiglet, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

// Rollback implements MigrationStep.
func (*removeAppLabelPatcher) Rollback(ctx context.Context, client *kube.Client, odigosNs string) error {
	patchType, patchDataAutoScaler := getUnPatchRemoveAppLabel(resources.AutoScalerAppLabelValue)
	_, err := client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.AutoScalerDeploymentName, patchType, patchDataAutoScaler, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchInstrumentor := getUnPatchRemoveAppLabel(resources.InstrumentorAppLabelValue)
	_, err = client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.InstrumentorDeploymentName, patchType, patchInstrumentor, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchSchedular := getUnPatchRemoveAppLabel(resources.SchedulerAppLabelValue)
	_, err = client.AppsV1().Deployments(odigosNs).Patch(ctx, resources.SchedulerDeploymentName, patchType, patchSchedular, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	patchType, patchDataOdiglet := getUnPatchRemoveAppLabel(resources.OdigletAppLabelValue)
	_, err = client.AppsV1().DaemonSets(odigosNs).Patch(ctx, resources.OdigletDaemonSetName, patchType, patchDataOdiglet, metav1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
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
