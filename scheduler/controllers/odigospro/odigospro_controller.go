package odigospro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type odigossecretController struct {
	client.Client
}

type proConfig struct {
	Audience string   `json:"aud"`
	Expiry   string   `json:"exp"` // Use int64 for UNIX timestamp
	Profiles []string `json:"profiles,omitempty"`
}

// TODO: logger
func (r *odigossecretController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling OdigosProSecret")

	odigosNs := env.GetCurrentNamespace()
	proSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosProSecretName}, proSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			err = r.handleSecretDeletion(ctx)
		}
		return ctrl.Result{}, err
	}

	tokenString := proSecret.Data[k8sconsts.OdigosProSecretTokenKeyName]
	if tokenString == nil {
		return ctrl.Result{}, reconcile.TerminalError(fmt.Errorf("error: token not found in secret %s/%s", odigosNs, k8sconsts.OdigosProSecretName))
	}

	token, _, err := jwt.NewParser().ParseUnverified(string(tokenString), jwt.MapClaims{})
	if err != nil {
		return ctrl.Result{}, reconcile.TerminalError(fmt.Errorf("error: failed to parse JWT token: %w", err))
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctrl.Result{}, reconcile.TerminalError(fmt.Errorf("error: failed to parse JWT token claims"))
	}
	proConfig := proConfig{
		Audience: claims["aud"].(string),
		// TODO: what time format should be used?
		Expiry:   time.Unix(int64(claims["exp"].(float64)), 0).UTC().Format("02 January 2006 03:04:05 PM"),
		Profiles: toStringSlice(claims["profiles"]),
	}

	odigosDeploymentConfig := &corev1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosDeploymentConfigMapName}, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	err = r.updateOdigosDeploymentConfigMap(ctx, proConfig, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	return ctrl.Result{}, nil
}

func (r *odigossecretController) handleSecretDeletion(ctx context.Context) error {
	odigosDeploymentConfig := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosDeploymentConfigMapName}, odigosDeploymentConfig)
	if err != nil {
		return reconcile.TerminalError(err)
	}
	for key := range odigosDeploymentConfig.Data {
		if strings.HasPrefix(key, "onprem_token") {
			delete(odigosDeploymentConfig.Data, key)
		}
	}

	err = r.Client.Update(ctx, odigosDeploymentConfig)
	if err != nil {
		return reconcile.TerminalError(err)
	}
	return nil
}

// convert []interface{} to []string
func toStringSlice(input interface{}) []string {
	if items, ok := input.([]interface{}); ok {
		result := make([]string, len(items))
		for i, item := range items {
			result[i] = item.(string)
		}
		return result
	}
	return nil
}

func (r *odigossecretController) updateOdigosDeploymentConfigMap(ctx context.Context, clientConfig proConfig, odigosDeploymentConfig *corev1.ConfigMap) error {
	odigosDeploymentConfig.Data["ONPREM_TOKEN_AUDIANCE"] = clientConfig.Audience
	odigosDeploymentConfig.Data["ONPREM_TOKEN_EXPIRY"] = clientConfig.Expiry

	if len(clientConfig.Profiles) > 0 {
		odigosDeploymentConfig.Data["ONPREM_TOKEN_PROFILES"] = strings.Join(clientConfig.Profiles, ",")
	}

	err := r.Client.Update(ctx, odigosDeploymentConfig)
	if err != nil {
		//TODO: is it suite to return `ctrl.Result{RequeueAfter: time.Minute}, nil`?
		return err
	}

	return nil
}
