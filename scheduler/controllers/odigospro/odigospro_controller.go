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

// TODO: logger
func (r *odigossecretController) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	odigosNs := env.GetCurrentNamespace()

	odigosDeploymentConfig := &corev1.ConfigMap{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: odigosNs, Name: k8sconsts.OdigosDeploymentConfigMapName}, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, reconcile.TerminalError(err)
	}

	proSecret := &corev1.Secret{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: odigosNs, Name: k8sconsts.OdigosProSecretName}, proSecret)

	if apierrors.IsNotFound(err) {
		deleted := deleteProInfoFromConfigMap(odigosDeploymentConfig)
		if deleted {
			logger.V(0).Info("OdigosPro secret not found, deleting pro info from odigos deployment config")
		}
	} else {
		err := updateProInfoInConfigMap(odigosDeploymentConfig, proSecret)
		if err != nil {
			return ctrl.Result{}, reconcile.TerminalError(err)
		}
		logger.V(0).Info("Updated pro info in odigos deployment config")
	}

	err = r.Client.Update(ctx, odigosDeploymentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// delete the pro info and retrun if the info was deleted
func deleteProInfoFromConfigMap(odigosDeploymentConfig *corev1.ConfigMap) bool {
	_, foundAudiance := odigosDeploymentConfig.Data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey]
	delete(odigosDeploymentConfig.Data, k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey)
	delete(odigosDeploymentConfig.Data, k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey)
	delete(odigosDeploymentConfig.Data, k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey)
	return foundAudiance
}

func updateProInfoInConfigMap(odigosDeploymentConfig *corev1.ConfigMap, proSecret *corev1.Secret) error {
	tokenString := proSecret.Data[k8sconsts.OdigosProSecretTokenKeyName]
	if tokenString == nil {
		return fmt.Errorf("error: token not found in secret")
	}

	token, _, err := jwt.NewParser().ParseUnverified(string(tokenString), jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("error: failed to parse JWT token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("error: failed to parse JWT token claims")
	}

	audience, ok := claims["aud"].(string)
	if !ok {
		return fmt.Errorf("error: failed to parse JWT token audience")
	}

	expiry, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("error: failed to parse JWT token expiry")
	}

	profilesString, profilesExists, err := getProfilesString(claims)
	if err != nil {
		return err
	}

	odigosDeploymentConfig.Data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenAudKey] = audience
	odigosDeploymentConfig.Data[k8sconsts.OdigosDeploymentConfigMapOnPremTokenExpKey] = time.Unix(int64(expiry), 0).UTC().Format("02 Jan 2006 03:04:05 PM")
	if profilesExists {
		odigosDeploymentConfig.Data[k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey] = profilesString
	} else {
		delete(odigosDeploymentConfig.Data, k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey)
	}

	return nil
}

func getProfilesString(claims jwt.MapClaims) (string, bool, error) {
	profiles, ok := claims["profiles"]
	if !ok {
		return "", false, nil
	}

	profilesSlice, ok := profiles.([]interface{})
	if !ok {
		return "", false, fmt.Errorf("error: failed to parse JWT token profiles")
	}

	if len(profilesSlice) == 0 {
		return "", false, nil
	}

	profileStrings := make([]string, 0, len(profilesSlice))
	for _, profile := range profilesSlice {
		profileString, ok := profile.(string)
		if !ok {
			return "", false, fmt.Errorf("error: found JWT profile which is not a string")
		}
		profileStrings = append(profileStrings, profileString)
	}

	return strings.Join(profileStrings, ", "), true, nil
}
