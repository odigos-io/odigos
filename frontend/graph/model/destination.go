package model

import (
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GetDestinationTypesResponse struct {
	Categories []DestinationsCategory `json:"categories"`
}

type DestinationTypesCategoryItem struct {
	Type                    string           `json:"type"`
	DisplayName             string           `json:"display_name"`
	ImageUrl                string           `json:"image_url"`
	SupportedSignals        SupportedSignals `json:"supported_signals"`
	TestConnectionSupported bool             `json:"test_connection_supported"`
}

type SupportedSignals struct {
	Traces  ObservabilitySignalSupport `json:"traces"`
	Metrics ObservabilitySignalSupport `json:"metrics"`
	Logs    ObservabilitySignalSupport `json:"logs"`
}

type ObservabilitySignalSupport struct {
	Supported bool `json:"supported"`
}
type DestinationsCategory struct {
	Name  string                         `json:"name"`
	Items []DestinationTypesCategoryItem `json:"items"`
}

type ExportedSignals struct {
	Traces  bool `json:"traces"`
	Metrics bool `json:"metrics"`
	Logs    bool `json:"logs"`
}

type Destination struct {
	Id              string                       `json:"id"`
	Name            string                       `json:"name"`
	Type            common.DestinationType       `json:"type"`
	ExportedSignals ExportedSignals              `json:"signals"`
	Fields          string                       `json:"fields"`
	DestinationType DestinationTypesCategoryItem `json:"destination_type"`
	Conditions      []metav1.Condition           `json:"conditions,omitempty"`
}
