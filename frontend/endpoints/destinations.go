package endpoints

import (
	"github.com/gin-gonic/gin"
	"github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/common"
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

func GetDestinations(c *gin.Context, odigosna string) {
	dests, err := kube.DefaultClient.OdigosClient.Destinations(odigosna).List(c, metav1.ListOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	var resp []Destination
	for _, dest := range dests.Items {
		endpointDest := k8sDestinationToEndpointFormat(dest)
		resp = append(resp, endpointDest)
	}

	c.JSON(200, resp)
}

func GetDestinationByName(c *gin.Context, odigosns string) {
	destName := c.Param("name")
	destination, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Get(c, destName, metav1.GetOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*destination)
	c.JSON(200, resp)
}

func PersistDestination(c *gin.Context, odigosns string) {

	request := Destination{}
	if err := c.ShouldBindJSON(&request); err != nil {
		returnError(c, err)
		return
	}

	k8sDestination := v1alpha1.Destination{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: request.Name,
		},
		Spec:       v1alpha1.DestinationSpec{
			Type:      request.Type,
			Data:      request.Data,
			Signals:   exportedSignalsObjectToSlice(request.ExportedSignals),
		},
		Status:     v1alpha1.DestinationStatus{},
	}
	dest, err := kube.DefaultClient.OdigosClient.Destinations(odigosns).Create(c, &k8sDestination, metav1.CreateOptions{})
	if err != nil {
		returnError(c, err)
		return
	}

	resp := k8sDestinationToEndpointFormat(*dest)
	c.JSON(201, resp)
}

func k8sDestinationToEndpointFormat(k8sDest v1alpha1.Destination) Destination {
	destType := k8sDest.Spec.Type
	destName := k8sDest.Name

	return Destination{
		Name: destName,
		Type: destType,
		ExportedSignals: ExportedSignals{
			Traces: isSignalExported(k8sDest, common.TracesObservabilitySignal),
			Metrics: isSignalExported(k8sDest, common.MetricsObservabilitySignal),
			Logs: isSignalExported(k8sDest, common.LogsObservabilitySignal),
		},
		Data: k8sDest.Spec.Data,
	}
}

func isSignalExported(dest v1alpha1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}

func exportedSignalsObjectToSlice(signals ExportedSignals) []common.ObservabilitySignal {
	var resp []common.ObservabilitySignal
	if signals.Traces {
		resp = append(resp, common.TracesObservabilitySignal)
	}
	if signals.Metrics {
		resp = append(resp, common.MetricsObservabilitySignal)
	}
	if signals.Logs {
		resp = append(resp, common.LogsObservabilitySignal)
	}

	return resp
}
