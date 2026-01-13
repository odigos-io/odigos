package kube

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
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
