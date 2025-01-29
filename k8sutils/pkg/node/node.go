package node

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func AddLabelToNode(nodeName string, labelKey string, labelValue string) error {
	// Add odiglet installed label to node
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	patch := []byte(`{"metadata": {"labels": {"` + labelKey + `": "` + labelValue + `"}}}`)
	_, err = clientset.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
