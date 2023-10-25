package migrations

import (
	"context"

	"github.com/keyval-dev/odigos/cli/pkg/containers"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

// Set the image field for a deployment or daemonset
type setTemplateImagePatcher struct {
	Patcher

	resourceInterface dynamic.ResourceInterface
	objectName        string
	imageName         string
	targetImageTag    string
	sourceImageTag    string
	containerName     string
}

func NewSetImagePatcherDeployment(client *kube.Client, ns string, objectName string, imageName string, targetImageTag string, sourceImageTag string, containerName string) Patcher {
	return &setTemplateImagePatcher{
		resourceInterface: getResourceInterfaceDeployment(client, ns),
		objectName:        objectName,
		imageName:         imageName,
		targetImageTag:    targetImageTag,
		sourceImageTag:    sourceImageTag,
		containerName:     containerName,
	}
}

func NewSetImagePatcherDaemonSet(client *kube.Client, ns string, objectName string, imageName string, imageTag string, containerName string) Patcher {
	return &setTemplateImagePatcher{
		resourceInterface: getResourceInterfaceDaemonSet(client, ns),
		objectName:        objectName,
		imageName:         imageName,
		targetImageTag:    imageTag,
		containerName:     containerName,
	}
}

func (p *setTemplateImagePatcher) PatcherName() string {
	return "SetTemplateImage"
}

func (p *setTemplateImagePatcher) Patch(ctx context.Context) error {
	patchType, patchData := getPatchTemplateSpecImage(p.imageName, p.targetImageTag, p.containerName)
	_, err := p.resourceInterface.Patch(ctx, p.objectName, patchType, patchData, metav1.PatchOptions{})
	return err
}

func (p *setTemplateImagePatcher) UnPatch(ctx context.Context) error {
	patchType, patchData := getPatchTemplateSpecImage(p.imageName, p.sourceImageTag, p.containerName)
	_, err := p.resourceInterface.Patch(ctx, p.objectName, patchType, patchData, metav1.PatchOptions{})
	return err
}

func getPatchTemplateSpecImage(imageName string, imageNewTag string, containerName string) (k8stypes.PatchType, []byte) {
	newImage := containers.GetImageName(imageName, imageNewTag)
	jsonBytes := encodeJsonPatchDocument(jsonPatchDocument{
		{
			Op:    "test",
			Path:  "/spec/template/spec/containers/0/name",
			Value: containerName,
		},
		{
			Op:    "replace",
			Path:  "/spec/template/spec/containers/0/image",
			Value: newImage,
		},
	})
	return k8stypes.JSONPatchType, jsonBytes
}
