package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/services"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AddClusterInfoDetails struct {
	ClusterAttributes []model.ClusterInfo `json:"clusterAttributes"`
}

func CreateAddClusterInfo(ctx context.Context, action model.ActionInput) (model.Action, error) {
	odigosns := consts.DefaultOdigosNamespace

	var details AddClusterInfoDetails
	err := json.Unmarshal([]byte(action.Details), &details)
	if err != nil {
		return nil, fmt.Errorf("invalid details for AddClusterInfo: %v", err)
	}

	signals, err := services.ConvertSignals(action.Signals)
	if err != nil {
		return nil, fmt.Errorf("failed to convert signals: %v", err)
	}

	clusterAttributes := make([]v1alpha1.OtelAttributeWithValue, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		clusterAttributes[i] = v1alpha1.OtelAttributeWithValue{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	addClusterInfoAction := &v1alpha1.AddClusterInfo{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "aci-",
		},
		Spec: v1alpha1.AddClusterInfoSpec{
			ActionName:        services.DerefString(action.Name),
			Notes:             services.DerefString(action.Notes),
			Disabled:          action.Disable,
			Signals:           signals,
			ClusterAttributes: clusterAttributes,
		},
	}

	generatedAction, err := kube.DefaultClient.ActionsClient.AddClusterInfos(odigosns).Create(ctx, addClusterInfoAction, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create AddClusterInfo: %v", err)
	}

	spec := make([]*model.ClusterInfo, len(details.ClusterAttributes))
	for i, attr := range details.ClusterAttributes {
		spec[i] = &model.ClusterInfo{
			AttributeName:        attr.AttributeName,
			AttributeStringValue: attr.AttributeStringValue,
		}
	}

	response := &model.AddClusterInfoAction{
		ID:      generatedAction.Name,
		Type:    "AddClusterInfo",
		Name:    action.Name,
		Notes:   action.Notes,
		Disable: action.Disable,
		Signals: action.Signals,
		Spec:    spec,
	}

	return response, nil
}
