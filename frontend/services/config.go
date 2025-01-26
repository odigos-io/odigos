package services

import (
	"context"
	"log"

	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InstallationStatus string

const (
	NewInstallation InstallationStatus = "NEW"
	AppsSelected    InstallationStatus = "APPS_SELECTED"
	Finished        InstallationStatus = "FINISHED"
)

type GetConfigResponse struct {
	Installation InstallationStatus `json:"installation"`
}

func GetConfig(ctx context.Context) GetConfigResponse {
	var response GetConfigResponse

	if !isSourceCreated(ctx) {
		response.Installation = NewInstallation
	} else if !isDestinationConnected(ctx) {
		response.Installation = AppsSelected
	} else {
		response.Installation = Finished
	}

	return response
}

func isSourceCreated(ctx context.Context) bool {
	nsList, err := kube.DefaultClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing namespaces: %v\n", err)
		return false
	}

	for _, ns := range nsList.Items {
		sourceList, err := kube.DefaultClient.OdigosClient.Sources(ns.Namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			log.Printf("Error listing sources: %v\n", err)
			return false
		}

		if len(sourceList.Items) > 0 {
			return true
		}
	}

	return false
}

func isDestinationConnected(ctx context.Context) bool {
	ns := env.GetCurrentNamespace()

	dests, err := kube.DefaultClient.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Printf("Error listing destinations: %v\n", err)
		return false
	}

	return len(dests.Items) > 0
}
