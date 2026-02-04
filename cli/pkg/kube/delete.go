package kube

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func DeleteDeploymentsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.AppsV1().Deployments(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteServicesByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.CoreV1().Services(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteRolesByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.RbacV1().Roles(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.RbacV1().Roles(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteRoleBindingsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.RbacV1().RoleBindings(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.RbacV1().RoleBindings(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteServiceAccountsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.CoreV1().ServiceAccounts(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.CoreV1().ServiceAccounts(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteSecretsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.CoreV1().Secrets(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteConfigMapsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.CoreV1().ConfigMaps(ns).Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteHPAsByLabel(ctx context.Context, client *Client, ns string, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()

	// HPA apiVersion differs by server version. Use the dynamic client and try all known served versions.
	// Deleting by GVR ensures we can clean up regardless of autoscaling API version.
	candidates := []schema.GroupVersionResource{
		{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
		{Group: "autoscaling", Version: "v2beta2", Resource: "horizontalpodautoscalers"},
		{Group: "autoscaling", Version: "v2beta1", Resource: "horizontalpodautoscalers"},
	}

	var lastErr error
	for _, gvr := range candidates {
		list, err := client.Dynamic.Resource(gvr).Namespace(ns).List(ctx, metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			lastErr = err
			continue
		}
		for _, item := range list.Items {
			if e := client.Dynamic.Resource(gvr).Namespace(ns).Delete(ctx, item.GetName(), metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
				return e
			}
		}
		// If we managed to list successfully for a served version, we're done.
		return nil
	}

	// If none of the versions were served, ignore; otherwise return the last non-NotFound error.
	return lastErr
}

func DeleteClusterRolesByLabel(ctx context.Context, client *Client, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.RbacV1().ClusterRoles().Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteClusterRoleBindingsByLabel(ctx context.Context, client *Client, labelKey string) error {
	selector := k8slabels.SelectorFromSet(map[string]string{labelKey: k8sconsts.OdigosSystemLabelValue}).String()
	list, err := client.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return err
	}
	for _, item := range list.Items {
		if e := client.RbacV1().ClusterRoleBindings().Delete(ctx, item.Name, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
			return e
		}
	}
	return nil
}

func DeleteCentralTokenSecret(ctx context.Context, client *Client, ns string) error {
	_, err := client.CoreV1().Secrets(ns).Get(ctx, k8sconsts.OdigosCentralSecretName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if e := client.CoreV1().Secrets(ns).Delete(ctx, k8sconsts.OdigosCentralSecretName, metav1.DeleteOptions{}); e != nil && !apierrors.IsNotFound(e) {
		return e
	}
	return nil
}

func NamespaceHasLabel(ctx context.Context, client *Client, ns string, labelKey string) (bool, error) {
	obj, err := client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if obj.Labels != nil {
		val, exists := obj.Labels[labelKey]
		return exists && val == k8sconsts.OdigosSystemLabelValue, nil
	}
	return false, nil
}
