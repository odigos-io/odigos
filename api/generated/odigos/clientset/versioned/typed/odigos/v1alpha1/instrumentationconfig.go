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
	"context"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	scheme "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/scheme"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// InstrumentationConfigsGetter has a method to return a InstrumentationConfigInterface.
// A group's client should implement this interface.
type InstrumentationConfigsGetter interface {
	InstrumentationConfigs(namespace string) InstrumentationConfigInterface
}

// InstrumentationConfigInterface has methods to work with InstrumentationConfig resources.
type InstrumentationConfigInterface interface {
	Create(ctx context.Context, instrumentationConfig *v1alpha1.InstrumentationConfig, opts v1.CreateOptions) (*v1alpha1.InstrumentationConfig, error)
	Update(ctx context.Context, instrumentationConfig *v1alpha1.InstrumentationConfig, opts v1.UpdateOptions) (*v1alpha1.InstrumentationConfig, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.InstrumentationConfig, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.InstrumentationConfigList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.InstrumentationConfig, err error)
	Apply(ctx context.Context, instrumentationConfig *odigosv1alpha1.InstrumentationConfigApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentationConfig, err error)
	InstrumentationConfigExpansion
}

// instrumentationConfigs implements InstrumentationConfigInterface
type instrumentationConfigs struct {
	*gentype.ClientWithListAndApply[*v1alpha1.InstrumentationConfig, *v1alpha1.InstrumentationConfigList, *odigosv1alpha1.InstrumentationConfigApplyConfiguration]
}

// newInstrumentationConfigs returns a InstrumentationConfigs
func newInstrumentationConfigs(c *OdigosV1alpha1Client, namespace string) *instrumentationConfigs {
	return &instrumentationConfigs{
		gentype.NewClientWithListAndApply[*v1alpha1.InstrumentationConfig, *v1alpha1.InstrumentationConfigList, *odigosv1alpha1.InstrumentationConfigApplyConfiguration](
			"instrumentationconfigs",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.InstrumentationConfig { return &v1alpha1.InstrumentationConfig{} },
			func() *v1alpha1.InstrumentationConfigList { return &v1alpha1.InstrumentationConfigList{} }),
	}
}
