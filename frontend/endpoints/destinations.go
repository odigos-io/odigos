package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/frontend/destinations"
)

type GetDestinationsResponse struct {
	Categories []DestinationsCategory `json:"categories"`
}

type DestinationsCategory struct {
	Name  string                     `json:"name"`
	Items []DestinationsCategoryItem `json:"items"`
}

type DestinationsCategoryItem struct {
	Type             string           `json:"type"`
	DisplayName      string           `json:"display_name"`
	ImageUrl         string           `json:"image_url"`
	SupportedSignals SupportedSignals `json:"supported_signals"`
}

type SupportedSignals struct {
	Traces  ObservabilitySignal `json:"traces"`
	Metrics ObservabilitySignal `json:"metrics"`
	Logs    ObservabilitySignal `json:"logs"`
}

type ObservabilitySignal struct {
	Supported bool `json:"supported"`
}

func GetDestinations(c *gin.Context) {
	var resp GetDestinationsResponse
	itemsByCategory := make(map[string][]DestinationsCategoryItem)
	for _, dest := range destinations.Get() {
		item := DestinationsCategoryItem{
			Type:        dest.Metadata.Type,
			DisplayName: dest.Metadata.DisplayName,
			ImageUrl:    GetImageURL(dest.Spec.Image),
			SupportedSignals: SupportedSignals{
				Traces: ObservabilitySignal{
					Supported: dest.Spec.Signals.Traces.Supported,
				},
				Metrics: ObservabilitySignal{
					Supported: dest.Spec.Signals.Metrics.Supported,
				},
				Logs: ObservabilitySignal{
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

func GetDestinationDetails(c *gin.Context) {
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
