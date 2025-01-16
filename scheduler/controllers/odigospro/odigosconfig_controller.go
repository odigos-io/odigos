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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type odigosConfigController struct {
	client.Client
	Scheme *runtime.Scheme
}

type ClientProConfig struct {
	Audience string   `json:"aud"`
	Expiry   string   `json:"exp"` // Use int64 for UNIX timestamp
	Profiles []string `json:"profiles,omitempty"`
}

// TODO: logger
func (r *odigosConfigController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling OdigosProConfig")

	proSecret := &corev1.Secret{}
	odigosDeploymentConfig := &corev1.ConfigMap{}
	odigosNs := env.GetCurrentNamespace()
	clientTokenConfig := &ClientProConfig{}

	err := r.Client.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosProSecretName}, proSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return r.handleSecretDeletion(ctx, r.Client, odigosDeploymentConfig)
		}
		return ctrl.Result{}, nil
	}

	tokenString := proSecret.Data["odigos-onprem-token"]
	// TODO: what if not found?

	token, _, err := jwt.NewParser().ParseUnverified(string(tokenString), jwt.MapClaims{})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse JWT token: %w", err)
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		*clientTokenConfig = ClientProConfig{
			Audience: claims["aud"].(string),
			// TODO: what time format should be used?
			Expiry:   time.Unix(int64(claims["exp"].(float64)), 0).UTC().Format("02/01/2006 03:04:05 PM"),
			Profiles: toStringSlice(claims["profiles"]),
		}
	} else {
		// TODO: is there anything to do in case of jwt parsing error?
	}

	err = r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosDeploymentConfigMapName}, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	r.updateOdigosDeploymentConfigMap(clientTokenConfig, odigosDeploymentConfig)

	err = r.Client.Update(ctx, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *odigosConfigController) handleSecretDeletion(ctx context.Context, client client.Client, odigosDeploymentConfig *corev1.ConfigMap) (ctrl.Result, error) {
	client.Get(ctx, types.NamespacedName{Namespace: env.GetCurrentNamespace(), Name: k8sconsts.OdigosDeploymentConfigMapName}, odigosDeploymentConfig)

	delete(odigosDeploymentConfig.Data, "audience")
	delete(odigosDeploymentConfig.Data, "expiry")
	delete(odigosDeploymentConfig.Data, "profiles")

	err := client.Update(ctx, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

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

func (r *odigosConfigController) updateOdigosDeploymentConfigMap(clientConfig *ClientProConfig, odigosDeploymentConfig *corev1.ConfigMap) {
	odigosDeploymentConfig.Data["audience"] = clientConfig.Audience
	odigosDeploymentConfig.Data["expiry"] = clientConfig.Expiry

	if len(clientConfig.Profiles) > 0 {
		odigosDeploymentConfig.Data["profiles"] = strings.Join(clientConfig.Profiles, ",")
	}
}
