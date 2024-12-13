package describe_utils

import (
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

func ConvertEntityPropertyToGQL(prop *properties.EntityProperty) *model.EntityProperty {
	if prop == nil {
		return nil
	}

	var value string
	if strValue, ok := prop.Value.(string); ok {
		value = strValue
	} else {
		value = fmt.Sprintf("%v", prop.Value)
	}

	var status *string
	if prop.Status != "" {
		statusStr := string(prop.Status)
		status = &statusStr
	}

	var explain *string
	if prop.Explain != "" {
		explain = &prop.Explain
	}

	return &model.EntityProperty{
		Name:    prop.Name,
		Value:   value,
		Status:  status,
		Explain: explain,
	}
}
