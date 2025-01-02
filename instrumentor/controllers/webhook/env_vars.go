package webhook

import corev1 "k8s.io/api/core/v1"

type modificationFunc func(origVal string, newVal string) string

var (
	Upsert modificationFunc = func(origVal string, newVal string) string {
		return newVal
	}

	AppendWithSpace modificationFunc = func(origVal string, newVal string) string {
		return origVal + " " + newVal
	}

	AppendWithComma modificationFunc = func(origVal string, newVal string) string {
		return origVal + "," + newVal
	}
)

type envVarModification struct {
	Value     string
	ValueFrom *corev1.EnvVarSource
	Action    modificationFunc
}

// modifyEnvVars modifies the environment variables of a container based on the modifications map.
// The modifications map is a map of env var keys to modification objects (value and action).
func (p *PodsWebhook) modifyEnvVars(original []corev1.EnvVar, modifications map[string]envVarModification) []corev1.EnvVar {
	remainingModifications := make(map[string]struct{}, len(modifications))
	for k := range modifications {
		remainingModifications[k] = struct{}{}
	}

	var result []corev1.EnvVar
	for _, envVar := range original {
		if modification, ok := modifications[envVar.Name]; ok {
			if modification.ValueFrom != nil {
				result = append(result, corev1.EnvVar{Name: envVar.Name, ValueFrom: modification.ValueFrom})
			} else {
				val := modification.Value
				if modification.Action != nil {
					val = modification.Action(envVar.Value, modification.Value)
				}

				result = append(result, corev1.EnvVar{Name: envVar.Name, Value: val})
			}
			delete(remainingModifications, envVar.Name)
		} else {
			result = append(result, envVar)
		}
	}

	for k := range remainingModifications {
		details := modifications[k]
		if details.ValueFrom != nil {
			result = append(result, corev1.EnvVar{Name: k, ValueFrom: details.ValueFrom})
		} else {
			result = append(result, corev1.EnvVar{Name: k, Value: details.Value})
		}
	}

	return result
}
