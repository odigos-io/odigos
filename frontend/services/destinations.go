package services

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func GetDestinationTypes() model.GetDestinationTypesResponse {
	var resp model.GetDestinationTypesResponse
	itemsByCategory := make(map[string][]model.DestinationTypesCategoryItem)
	for _, destConfig := range destinations.Get() {
		item := DestinationTypeConfigToCategoryItem(destConfig)
		itemsByCategory[destConfig.Metadata.Category] = append(itemsByCategory[destConfig.Metadata.Category], item)
	}

	for category, items := range itemsByCategory {
		resp.Categories = append(resp.Categories, model.DestinationsCategory{
			Name:  category,
			Items: items,
		})

	}

	return resp

}

func DestinationTypeConfigToCategoryItem(destConfig destinations.Destination) model.DestinationTypesCategoryItem {

	return model.DestinationTypesCategoryItem{
		Type:                    common.DestinationType(destConfig.Metadata.Type),
		DisplayName:             destConfig.Metadata.DisplayName,
		ImageUrl:                GetImageURL(destConfig.Spec.Image),
		TestConnectionSupported: destConfig.Spec.TestConnectionSupported,
		SupportedSignals: model.SupportedSignals{
			Traces: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Traces.Supported,
			},
			Metrics: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Metrics.Supported,
			},
			Logs: model.ObservabilitySignalSupport{
				Supported: destConfig.Spec.Signals.Logs.Supported,
			},
		},
	}

}

func GetDestinationTypeConfig(destType common.DestinationType) (*destinations.Destination, error) {
	for _, dest := range destinations.Get() {
		if dest.Metadata.Type == destType {
			return &dest, nil
		}
	}

	return nil, fmt.Errorf("destination type %s not found", destType)
}
