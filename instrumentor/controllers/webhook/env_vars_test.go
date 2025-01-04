package webhook

import (
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestModifyEnvVars(t *testing.T) {
	wh := &PodsWebhook{}

	tests := []struct {
		name          string
		original      []corev1.EnvVar
		modifications map[string]envVarModification
		expected      []corev1.EnvVar
	}{
		{
			name: "Modify existing environment variables",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {Value: "new_value1"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Add new environment variables",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR2": {Value: "value2"},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Apply custom action to existing variable",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {Value: "suffix", Action: func(currentValue, newValue string) string {
					return currentValue + newValue
				}},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1suffix"},
			},
		},
		{
			name: "Upsert modification",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {Value: "new_value1", Action: Upsert},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "new_value1"},
			},
		},
		{
			name: "Append with space",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {Value: "suffix", Action: AppendWithSpace},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1 suffix"},
			},
		},
		{
			name: "Append with comma",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {Value: "suffix", Action: AppendWithComma},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1,suffix"},
			},
		},
		{
			name: "No modifications",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
		},
		{
			name: "Modify non-existing key",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR2": {Value: "value2", Action: Upsert},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Add non-existing key with append",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR2": {Value: "value2", Action: AppendWithSpace},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", Value: "value2"},
			},
		},
		{
			name: "Add value with ValueFrom",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR2": {ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}, Action: Upsert},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
				{Name: "VAR2", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			},
		},
		{
			name: "Modify existing key with ValueFrom",
			original: []corev1.EnvVar{
				{Name: "VAR1", Value: "value1"},
			},
			modifications: map[string]envVarModification{
				"VAR1": {ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}, Action: Upsert},
			},
			expected: []corev1.EnvVar{
				{Name: "VAR1", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wh.modifyEnvVars(tt.original, tt.modifications)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
