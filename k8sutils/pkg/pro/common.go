package pro

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/odigosauth"
)

func UpdateOdigosToken(ctx context.Context, client kubernetes.Interface, namespace string, onPremToken string) error {
	if _, err := odigosauth.ValidateToken(onPremToken); err != nil {
		return err
	}
	if err := updateSecretToken(ctx, client, namespace, onPremToken); err != nil {
		return fmt.Errorf("failed to update secret token: %w", err)
	}
	if err := odigletRolloutTrigger(ctx, client, namespace); err != nil {
		return fmt.Errorf("failed to trigger odiglet rollout: %w", err)
	}
	return nil
}

func updateSecretToken(ctx context.Context, client kubernetes.Interface, namespace string, onPremToken string) error {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, k8sconsts.OdigosProSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("tokens are not available in the community version of Odigos. Please contact Odigos team to inquire about pro version")
		}
		return err
	}
	secret.Data[k8sconsts.OdigosOnpremTokenSecretKey] = []byte(onPremToken)

	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func odigletRolloutTrigger(ctx context.Context, client kubernetes.Interface, namespace string) error {
	daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(ctx, "odiglet", metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("\033[31mERROR\033[0m failed to get odiglet DaemonSet in namespace %s: %v", namespace, err)
	}

	// Modify the DaemonSet spec.template to trigger a rollout
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.Annotations[odigosconsts.RolloutTriggerAnnotation] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		command := fmt.Sprintf("kubectl rollout restart daemonset odiglet -n %s", daemonSet.Namespace)
		return fmt.Errorf("failed to restart Odiglets: %w. To trigger a restart manually, run the following command: %s", err, command)
	}
	return nil
}

type TokenPayload struct {
	OnpremToken string `json:"token"`
}
