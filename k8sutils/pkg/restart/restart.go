package restart

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func RestartDeployment(ctx context.Context, client kubernetes.Interface, namespace string, deploymentName string) error {
	_, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":%q}}}}}`,
		time.Now().Format(time.RFC3339))

	_, err = client.AppsV1().Deployments(namespace).Patch(
		ctx,
		deploymentName,
		k8stypes.StrategicMergePatchType,
		[]byte(patch),
		metav1.PatchOptions{},
	)
	return err
}
