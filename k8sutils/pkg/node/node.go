package node

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func AddLabelToNode(clientset *kubernetes.Clientset, nodeName string, labelKey string, labelValue string) error {
	patch := []byte(`{"metadata": {"labels": {"` + labelKey + `": "` + labelValue + `"}}}`)
	_, err := clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

func RemoveLabelFromNode(clientset *kubernetes.Clientset, nodeName string, labelKey string) error {
	patch := []byte(`{"metadata": {"labels": {"` + labelKey + `": null}}}`)
	_, err := clientset.CoreV1().Nodes().Patch(context.TODO(), nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
