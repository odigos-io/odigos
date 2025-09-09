/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	actionv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

func TestActionsValidator_ValidateCreate(t *testing.T) {
	// Set up test environment
	os.Setenv("POD_NAMESPACE", "odigos-system")
	defer os.Unsetenv("POD_NAMESPACE")

	validator := &ActionsValidator{}

	tests := []struct {
		name        string
		action      *odigosv1.Action
		expectError bool
		errorType   field.ErrorType
		errorField  string
	}{
		{
			name: "valid AddClusterInfo action",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-add-cluster-info",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid DeleteAttribute action",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-delete-attribute",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					DeleteAttribute: &actionv1.DeleteAttributeConfig{
						AttributeNamesToDelete: []string{"test.key"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid RenameAttribute action",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-rename-attribute",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					RenameAttribute: &actionv1.RenameAttributeConfig{
						Renames: map[string]string{
							"old.key": "new.key",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid PiiMasking action",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-pii-masking",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					PiiMasking: &actionv1.PiiMaskingConfig{
						PiiCategories: []actionv1.PiiCategory{actionv1.CreditCardMasking},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid K8sAttributes action",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-k8s-attributes",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						CollectContainerAttributes: true,
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid namespace",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			expectError: true,
			errorType:   field.ErrorTypeInvalid,
			errorField:  "namespace",
		},
		{
			name: "no action config",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					// No action config specified
				},
			},
			expectError: true,
			errorType:   field.ErrorTypeRequired,
			errorField:  "spec",
		},
		{
			name: "multiple action configs",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
					DeleteAttribute: &actionv1.DeleteAttributeConfig{
						AttributeNamesToDelete: []string{"test.key"},
					},
				},
			},
			expectError: true,
			errorType:   field.ErrorTypeInvalid,
			errorField:  "spec.addClusterInfo", // Should have multiple field errors
		},
		{
			name: "non-Action object",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := validator.ValidateCreate(context.Background(), tt.action)

			if tt.expectError {
				assert.Error(t, err)
				// Note: The actual error type from ValidateCreate is *apierrors.StatusError
				// which wraps the field.ErrorList, so we can't directly access field errors here
			} else {
				assert.NoError(t, err)
			}
			assert.Empty(t, warnings)
		})
	}
}

func TestActionsValidator_ValidateUpdate(t *testing.T) {
	// Set up test environment
	os.Setenv("POD_NAMESPACE", "odigos-system")
	defer os.Unsetenv("POD_NAMESPACE")

	validator := &ActionsValidator{}

	tests := []struct {
		name        string
		oldAction   *odigosv1.Action
		newAction   *odigosv1.Action
		expectError bool
	}{
		{
			name: "valid update",
			oldAction: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("old-cluster"),
							},
						},
					},
				},
			},
			newAction: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("new-cluster"),
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid update - wrong namespace",
			oldAction: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			newAction: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "default", // Wrong namespace
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := validator.ValidateUpdate(context.Background(), tt.oldAction, tt.newAction)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Empty(t, warnings)
		})
	}
}

func TestActionsValidator_ValidateAction(t *testing.T) {
	// Set up test environment
	os.Setenv("POD_NAMESPACE", "odigos-system")
	defer os.Unsetenv("POD_NAMESPACE")

	validator := &ActionsValidator{}

	tests := []struct {
		name        string
		action      *odigosv1.Action
		expectError bool
		errorCount  int
		errorTypes  []field.ErrorType
		errorFields []string
	}{
		{
			name: "valid action with correct namespace",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid namespace",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "wrong-namespace",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
			expectError: true,
			errorCount:  1,
			errorTypes:  []field.ErrorType{field.ErrorTypeInvalid},
			errorFields: []string{"namespace"},
		},
		{
			name: "no action config",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					// No action configs
				},
			},
			expectError: true,
			errorCount:  1,
			errorTypes:  []field.ErrorType{field.ErrorTypeRequired},
			errorFields: []string{"spec"},
		},
		{
			name: "multiple action configs",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
					DeleteAttribute: &actionv1.DeleteAttributeConfig{
						AttributeNamesToDelete: []string{"test.key"},
					},
				},
			},
			expectError: true,
			errorCount:  2,
			errorTypes:  []field.ErrorType{field.ErrorTypeInvalid, field.ErrorTypeInvalid},
			errorFields: []string{"spec.addClusterInfo", "spec.deleteAttribute"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateAction(context.Background(), tt.action)

			if tt.expectError {
				assert.NotEmpty(t, errors)
				assert.Equal(t, tt.errorCount, len(errors))

				if len(tt.errorTypes) > 0 {
					for i, expectedType := range tt.errorTypes {
						if i < len(errors) {
							assert.Equal(t, expectedType, errors[i].Type)
						}
					}
				}

				if len(tt.errorFields) > 0 {
					actualFields := make([]string, len(errors))
					for i, err := range errors {
						actualFields[i] = err.Field
					}
					for _, expectedField := range tt.errorFields {
						assert.Contains(t, actualFields, expectedField)
					}
				}
			} else {
				assert.Empty(t, errors)
			}
		})
	}
}

func TestActionsValidator_ValidateAction_AllActionTypes(t *testing.T) {
	// Set up test environment
	os.Setenv("POD_NAMESPACE", "odigos-system")
	defer os.Unsetenv("POD_NAMESPACE")

	validator := &ActionsValidator{}

	tests := []struct {
		name   string
		action *odigosv1.Action
	}{
		{
			name: "AddClusterInfo",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action-1",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action-1",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					AddClusterInfo: &actionv1.AddClusterInfoConfig{
						ClusterAttributes: []actionv1.OtelAttributeWithValue{
							{
								AttributeName:        "cluster.name",
								AttributeStringValue: stringPtr("test-cluster"),
							},
						},
					},
				},
			},
		},
		{
			name: "DeleteAttribute",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action-2",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action-2",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					DeleteAttribute: &actionv1.DeleteAttributeConfig{
						AttributeNamesToDelete: []string{"test.key"},
					},
				},
			},
		},
		{
			name: "RenameAttribute",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action-3",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action-3",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					RenameAttribute: &actionv1.RenameAttributeConfig{
						Renames: map[string]string{
							"old.key": "new.key",
						},
					},
				},
			},
		},
		{
			name: "PiiMasking",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action-4",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action-4",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					PiiMasking: &actionv1.PiiMaskingConfig{
						PiiCategories: []actionv1.PiiCategory{actionv1.CreditCardMasking},
					},
				},
			},
		},
		{
			name: "K8sAttributes",
			action: &odigosv1.Action{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-action-5",
					Namespace: "odigos-system",
				},
				Spec: odigosv1.ActionSpec{
					ActionName: "test-action-5",
					Signals:    []common.ObservabilitySignal{common.TracesObservabilitySignal},
					K8sAttributes: &actionv1.K8sAttributesConfig{
						CollectContainerAttributes: true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.validateAction(context.Background(), tt.action)
			assert.Empty(t, errors, "Action type %s should be valid", tt.name)
		})
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
