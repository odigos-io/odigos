/*
Copyright 2022.

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
// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/actions/v1alpha1"
	internal "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/internal"
	runtime "k8s.io/apimachinery/pkg/runtime"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	testing "k8s.io/client-go/testing"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=actions, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithKind("AddClusterInfo"):
		return &actionsv1alpha1.AddClusterInfoApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AddClusterInfoSpec"):
		return &actionsv1alpha1.AddClusterInfoSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("AddClusterInfoStatus"):
		return &actionsv1alpha1.AddClusterInfoStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("DeleteAttribute"):
		return &actionsv1alpha1.DeleteAttributeApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("DeleteAttributeSpec"):
		return &actionsv1alpha1.DeleteAttributeSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("DeleteAttributeStatus"):
		return &actionsv1alpha1.DeleteAttributeStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ErrorSampler"):
		return &actionsv1alpha1.ErrorSamplerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ErrorSamplerSpec"):
		return &actionsv1alpha1.ErrorSamplerSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ErrorSamplerStatus"):
		return &actionsv1alpha1.ErrorSamplerStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("HttpRouteFilter"):
		return &actionsv1alpha1.HttpRouteFilterApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("LatencySampler"):
		return &actionsv1alpha1.LatencySamplerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("LatencySamplerSpec"):
		return &actionsv1alpha1.LatencySamplerSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("LatencySamplerStatus"):
		return &actionsv1alpha1.LatencySamplerStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("OtelAttributeWithValue"):
		return &actionsv1alpha1.OtelAttributeWithValueApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("PiiMasking"):
		return &actionsv1alpha1.PiiMaskingApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("PiiMaskingSpec"):
		return &actionsv1alpha1.PiiMaskingSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("PiiMaskingStatus"):
		return &actionsv1alpha1.PiiMaskingStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ProbabilisticSampler"):
		return &actionsv1alpha1.ProbabilisticSamplerApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ProbabilisticSamplerSpec"):
		return &actionsv1alpha1.ProbabilisticSamplerSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("ProbabilisticSamplerStatus"):
		return &actionsv1alpha1.ProbabilisticSamplerStatusApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RenameAttribute"):
		return &actionsv1alpha1.RenameAttributeApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RenameAttributeSpec"):
		return &actionsv1alpha1.RenameAttributeSpecApplyConfiguration{}
	case v1alpha1.SchemeGroupVersion.WithKind("RenameAttributeStatus"):
		return &actionsv1alpha1.RenameAttributeStatusApplyConfiguration{}

	}
	return nil
}

func NewTypeConverter(scheme *runtime.Scheme) *testing.TypeConverter {
	return &testing.TypeConverter{Scheme: scheme, TypeResolver: internal.Parser()}
}