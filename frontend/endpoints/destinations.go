package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/common/utils"
	"github.com/keyval-dev/odigos/frontend/destinations"
	"github.com/keyval-dev/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetDestinationTypesResponse struct {
	Categories []DestinationsCategory `json:"categories"`
}

type DestinationsCategory struct {
	Name  string                     `json:"name"`
	Items []DestinationTypesCategoryItem `json:"items"`
}

type DestinationTypesCategoryItem struct {
	Type             string           `json:"type"`
	DisplayName      string           `json:"display_name"`
	ImageUrl         string           `json:"image_url"`
	SupportedSignals SupportedSignals `json:"supported_signals"`
}

type SupportedSignals struct {
	Traces  ObservabilitySignalSupport `json:"traces"`
	Metrics ObservabilitySignalSupport `json:"metrics"`
	Logs    ObservabilitySignalSupport `json:"logs"`
}

type ObservabilitySignalSupport struct {
	Supported bool `json:"supported"`
}

type ExportedSignals struct {
	Traces bool `json:"traces"`
	Metrics bool `json:"metrics"`
	Logs bool `json:"logs"`
}

type Destination struct {
	Name string `json:"name"`
	Type common.DestinationType `json:"type"`
	ExportedSignals ExportedSignals `json:"signals"`
	Data map[string]string `json:"data"`
}

func GetDestinationTypes(c *gin.Context) {
	var resp GetDestinationTypesResponse
	itemsByCategory := make(map[string][]DestinationTypesCategoryItem)
	for _, dest := range destinations.Get() {
		item := DestinationTypesCategoryItem{
			Type:        dest.Metadata.Type,
			DisplayName: dest.Metadata.DisplayName,
			ImageUrl:    GetImageURL(dest.Spec.Image),
			SupportedSignals: SupportedSignals{
				Traces: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Traces.Supported,
				},
				Metrics: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Metrics.Supported,
				},
				Logs: ObservabilitySignalSupport{
					Supported: dest.Spec.Signals.Logs.Supported,
				},
			},
		}

		itemsByCategory[dest.Metadata.Category] = append(itemsByCategory[dest.Metadata.Category], item)
	}

	for category, items := range itemsByCategory {
		resp.Categories = append(resp.Categories, DestinationsCategory{
			Name:  category,
			Items: items,
		})
	}

	c.JSON(200, resp)
}

type GetDestinationDetailsResponse struct {
	Fields []Field `json:"fields"`
}

type Field struct {
	Name                string                 `json:"name"`
	DisplayName         string                 `json:"display_name"`
	ComponentType       string                 `json:"component_type"`
	ComponentProperties map[string]interface{} `json:"component_properties"`
	VideoUrl            string                 `json:"video_url"`
}

func GetDestinationTypeDetails(c *gin.Context) {
	destType := c.Param("type")
	for _, dest := range destinations.Get() {
		if dest.Metadata.Type == destType {
			var resp GetDestinationDetailsResponse
			for _, field := range dest.Spec.Fields {
				resp.Fields = append(resp.Fields, Field{
					Name:                field.Name,
					DisplayName:         field.DisplayName,
					ComponentType:       field.ComponentType,
					ComponentProperties: field.ComponentProps,
					VideoUrl:            field.VideoURL,
				})
			}

			c.JSON(200, resp)
			return
		}
	}

	c.JSON(404, gin.H{
		"error": "destination not found",
	})
}

func GetDestinations(c *gin.Context) {
	currentns := utils.GetCurrentNamespace()
	dests, err := kube.DefaultClient.OdigosClient.Destinations(currentns).List(c, metav1.ListOptions{})
	if err != nil {
		c.JSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	var resp []Destination
	for _, dest := range dests.Items {
		destType := dest.Spec.Type
		destName := dest.Name

		resp = append(resp, Destination{
			Name: destName,
			Type: destType,
			ExportedSignals: ExportedSignals{
				Traces: isSignalExported(dest, common.TracesObservabilitySignal),
				Metrics: isSignalExported(dest, common.MetricsObservabilitySignal),
				Logs: isSignalExported(dest, common.LogsObservabilitySignal),
			},
			Data: dest.Spec.Data,
		})
	}

	c.JSON(200, resp)
}

func isSignalExported(dest v1alpha1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}