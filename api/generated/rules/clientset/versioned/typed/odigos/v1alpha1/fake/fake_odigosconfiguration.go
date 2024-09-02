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

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/rules/applyconfiguration/odigos/v1alpha1"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeOdigosConfigurations implements OdigosConfigurationInterface
type FakeOdigosConfigurations struct {
	Fake *FakeOdigosV1alpha1
	ns   string
}

var odigosconfigurationsResource = v1alpha1.SchemeGroupVersion.WithResource("odigosconfigurations")

var odigosconfigurationsKind = v1alpha1.SchemeGroupVersion.WithKind("OdigosConfiguration")

// Get takes name of the odigosConfiguration, and returns the corresponding odigosConfiguration object, and an error if there is any.
func (c *FakeOdigosConfigurations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.OdigosConfiguration, err error) {
	emptyResult := &v1alpha1.OdigosConfiguration{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(odigosconfigurationsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.OdigosConfiguration), err
}

// List takes label and field selectors, and returns the list of OdigosConfigurations that match those selectors.
func (c *FakeOdigosConfigurations) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.OdigosConfigurationList, err error) {
	emptyResult := &v1alpha1.OdigosConfigurationList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(odigosconfigurationsResource, odigosconfigurationsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.OdigosConfigurationList{ListMeta: obj.(*v1alpha1.OdigosConfigurationList).ListMeta}
	for _, item := range obj.(*v1alpha1.OdigosConfigurationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested odigosConfigurations.
func (c *FakeOdigosConfigurations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(odigosconfigurationsResource, c.ns, opts))

}

// Create takes the representation of a odigosConfiguration and creates it.  Returns the server's representation of the odigosConfiguration, and an error, if there is any.
func (c *FakeOdigosConfigurations) Create(ctx context.Context, odigosConfiguration *v1alpha1.OdigosConfiguration, opts v1.CreateOptions) (result *v1alpha1.OdigosConfiguration, err error) {
	emptyResult := &v1alpha1.OdigosConfiguration{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(odigosconfigurationsResource, c.ns, odigosConfiguration, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.OdigosConfiguration), err
}

// Update takes the representation of a odigosConfiguration and updates it. Returns the server's representation of the odigosConfiguration, and an error, if there is any.
func (c *FakeOdigosConfigurations) Update(ctx context.Context, odigosConfiguration *v1alpha1.OdigosConfiguration, opts v1.UpdateOptions) (result *v1alpha1.OdigosConfiguration, err error) {
	emptyResult := &v1alpha1.OdigosConfiguration{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(odigosconfigurationsResource, c.ns, odigosConfiguration, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.OdigosConfiguration), err
}

// Delete takes name of the odigosConfiguration and deletes it. Returns an error if one occurs.
func (c *FakeOdigosConfigurations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(odigosconfigurationsResource, c.ns, name, opts), &v1alpha1.OdigosConfiguration{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeOdigosConfigurations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(odigosconfigurationsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.OdigosConfigurationList{})
	return err
}

// Patch applies the patch and returns the patched odigosConfiguration.
func (c *FakeOdigosConfigurations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.OdigosConfiguration, err error) {
	emptyResult := &v1alpha1.OdigosConfiguration{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(odigosconfigurationsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.OdigosConfiguration), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied odigosConfiguration.
func (c *FakeOdigosConfigurations) Apply(ctx context.Context, odigosConfiguration *odigosv1alpha1.OdigosConfigurationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.OdigosConfiguration, err error) {
	if odigosConfiguration == nil {
		return nil, fmt.Errorf("odigosConfiguration provided to Apply must not be nil")
	}
	data, err := json.Marshal(odigosConfiguration)
	if err != nil {
		return nil, err
	}
	name := odigosConfiguration.Name
	if name == nil {
		return nil, fmt.Errorf("odigosConfiguration.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.OdigosConfiguration{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(odigosconfigurationsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.OdigosConfiguration), err
}
