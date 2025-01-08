package pro

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func UpdateOdigosToken(ctx context.Context, client *kube.Client, namespace string, onPremToken string) error {
	if err := updateSecretToken(ctx, client, namespace, onPremToken); err != nil {
		return fmt.Errorf("failed to update secret token: %w", err)
	}
	if err := odigletRolloutTrigger(ctx, client, namespace); err != nil {
		return fmt.Errorf("failed to trigger odiglet rollout: %w", err)
	}
	return nil
}

func updateSecretToken(ctx context.Context, client *kube.Client, namespace string, onPremToken string) error {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, consts.OdigosProSecretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("Tokens are not available in the open-source version of Odigos. Please contact Odigos team to inquire about pro version.")
		}
		return err
	}
	secret.Data[consts.OdigosOnpremTokenSecretKey] = []byte(onPremToken)

	_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func odigletRolloutTrigger(ctx context.Context, client *kube.Client, namespace string) error {
	daemonSet, err := client.AppsV1().DaemonSets(namespace).Get(ctx, "odiglet", metav1.GetOptions{})
	if err != nil {
		fmt.Errorf("\033[31mERROR\033[0m failed to get odiglet DaemonSet in namespace %s: %v\n", namespace, err)
	}

	// Modify the DaemonSet spec.template to trigger a rollout
	if daemonSet.Spec.Template.Annotations == nil {
		daemonSet.Spec.Template.Annotations = make(map[string]string)
	}
	daemonSet.Spec.Template.Annotations[odigosconsts.RolloutTriggerAnnotation] = time.Now().Format(time.RFC3339)

	_, err = client.AppsV1().DaemonSets(namespace).Update(ctx, daemonSet, metav1.UpdateOptions{})
	if err != nil {
		fmt.Printf("Failed to restart Odiglets. Reason: %s\n", err)
		fmt.Printf("To trigger a restart manually, run the following command:\n")
		fmt.Printf("kubectl rollout restart daemonset odiglet -n %s\n", daemonSet.Namespace)
	}
}
