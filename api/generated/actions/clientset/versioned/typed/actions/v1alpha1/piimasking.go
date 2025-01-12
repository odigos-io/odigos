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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"

	actionsv1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	applyconfigurationactionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/actions/v1alpha1"
	scheme "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// PiiMaskingsGetter has a method to return a PiiMaskingInterface.
// A group's client should implement this interface.
type PiiMaskingsGetter interface {
	PiiMaskings(namespace string) PiiMaskingInterface
}

// PiiMaskingInterface has methods to work with PiiMasking resources.
type PiiMaskingInterface interface {
	Create(ctx context.Context, piiMasking *actionsv1alpha1.PiiMasking, opts v1.CreateOptions) (*actionsv1alpha1.PiiMasking, error)
	Update(ctx context.Context, piiMasking *actionsv1alpha1.PiiMasking, opts v1.UpdateOptions) (*actionsv1alpha1.PiiMasking, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, piiMasking *actionsv1alpha1.PiiMasking, opts v1.UpdateOptions) (*actionsv1alpha1.PiiMasking, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*actionsv1alpha1.PiiMasking, error)
	List(ctx context.Context, opts v1.ListOptions) (*actionsv1alpha1.PiiMaskingList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *actionsv1alpha1.PiiMasking, err error)
	Apply(ctx context.Context, piiMasking *applyconfigurationactionsv1alpha1.PiiMaskingApplyConfiguration, opts v1.ApplyOptions) (result *actionsv1alpha1.PiiMasking, err error)
	// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
	ApplyStatus(ctx context.Context, piiMasking *applyconfigurationactionsv1alpha1.PiiMaskingApplyConfiguration, opts v1.ApplyOptions) (result *actionsv1alpha1.PiiMasking, err error)
	PiiMaskingExpansion
}

// piiMaskings implements PiiMaskingInterface
type piiMaskings struct {
	*gentype.ClientWithListAndApply[*actionsv1alpha1.PiiMasking, *actionsv1alpha1.PiiMaskingList, *applyconfigurationactionsv1alpha1.PiiMaskingApplyConfiguration]
}

// newPiiMaskings returns a PiiMaskings
func newPiiMaskings(c *ActionsV1alpha1Client, namespace string) *piiMaskings {
	return &piiMaskings{
		gentype.NewClientWithListAndApply[*actionsv1alpha1.PiiMasking, *actionsv1alpha1.PiiMaskingList, *applyconfigurationactionsv1alpha1.PiiMaskingApplyConfiguration](
			"piimaskings",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *actionsv1alpha1.PiiMasking { return &actionsv1alpha1.PiiMasking{} },
			func() *actionsv1alpha1.PiiMaskingList { return &actionsv1alpha1.PiiMaskingList{} },
		),
	}
}
