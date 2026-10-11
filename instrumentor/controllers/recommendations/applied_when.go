package recommendations

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmespath/go-jmespath"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

func getEffectiveConfigData(ctx context.Context, c client.Client) (any, error) {
	var configMap corev1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{
		Namespace: env.GetCurrentNamespace(),
		Name:      consts.OdigosEffectiveConfigName,
	}, &configMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, k8sutils.ErrOdigosEffectiveConfigNotFound
		}
		return nil, err
	}
	if configMap.Data == nil {
		return nil, fmt.Errorf("configMap.Data is nil")
	}
	cfgYAML := configMap.Data[consts.OdigosConfigurationFileName]
	if strings.TrimSpace(cfgYAML) == "" {
		return nil, fmt.Errorf("configMap.Data[consts.OdigosConfigurationFileName] is empty")
	}

	var data any
	if err := yaml.Unmarshal([]byte(cfgYAML), &data); err != nil {
		return nil, err
	}
	return data, nil
}

func evaluateAppliedWhen(cfgData any, actions []odigosv1.Action, rec recommendationcatalog.Recommendation) (bool, error) {
	if len(rec.AppliedWhen) == 0 {
		return false, nil
	}
	for _, check := range rec.AppliedWhen {
		holds, err := appliedWhenHolds(cfgData, actions, check)
		if err != nil {
			return false, err
		}
		if !holds {
			return false, nil
		}
	}
	return true, nil
}

func appliedWhenHolds(cfgData any, actions []odigosv1.Action, check recommendationcatalog.AppliedWhenCheck) (bool, error) {
	switch check.Type {
	case recommendationcatalog.AppliedWhenTypeEffectiveConfig:
		return effectiveConfigAppliedWhen(cfgData, check)
	case recommendationcatalog.AppliedWhenTypeActionExists:
		return actionExistsAppliedWhen(actions, check)
	default:
		return false, fmt.Errorf("unknown appliedWhen check type %q", check.Type)
	}
}

func actionExistsAppliedWhen(actions []odigosv1.Action, check recommendationcatalog.AppliedWhenCheck) (bool, error) {
	if strings.TrimSpace(check.ActionType) == "" {
		return false, fmt.Errorf("appliedWhen ActionExists actionType is empty")
	}

	for i := range actions {
		action := &actions[i]
		if action.Spec.Disabled {
			continue
		}
		if actionHasType(action, check.ActionType) {
			return true, nil
		}
	}
	return false, nil
}

func actionHasType(action *odigosv1.Action, actionType string) bool {
	field := reflect.ValueOf(action.Spec).FieldByName(actionType)
	return field.IsValid() && field.Kind() == reflect.Ptr && !field.IsNil()
}

func effectiveConfigAppliedWhen(cfgData any, check recommendationcatalog.AppliedWhenCheck) (bool, error) {
	if strings.TrimSpace(check.Expression) == "" {
		return false, fmt.Errorf("appliedWhen EffectiveConfig expression is empty")
	}

	result, err := jmespath.Search(check.Expression, cfgData)
	if err != nil {
		return false, fmt.Errorf("appliedWhen JMESPath %q: %w", check.Expression, err)
	}
	holds, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("appliedWhen JMESPath %q must return bool, got %T", check.Expression, result)
	}
	return holds, nil
}
