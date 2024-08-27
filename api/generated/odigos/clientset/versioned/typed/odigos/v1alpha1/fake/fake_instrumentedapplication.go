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

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeInstrumentedApplications implements InstrumentedApplicationInterface
type FakeInstrumentedApplications struct {
	Fake *FakeOdigosV1alpha1
	ns   string
}

var instrumentedapplicationsResource = v1alpha1.SchemeGroupVersion.WithResource("instrumentedapplications")

var instrumentedapplicationsKind = v1alpha1.SchemeGroupVersion.WithKind("InstrumentedApplication")

// Get takes name of the instrumentedApplication, and returns the corresponding instrumentedApplication object, and an error if there is any.
func (c *FakeInstrumentedApplications) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(instrumentedapplicationsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// List takes label and field selectors, and returns the list of InstrumentedApplications that match those selectors.
func (c *FakeInstrumentedApplications) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.InstrumentedApplicationList, err error) {
	emptyResult := &v1alpha1.InstrumentedApplicationList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(instrumentedapplicationsResource, instrumentedapplicationsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.InstrumentedApplicationList{ListMeta: obj.(*v1alpha1.InstrumentedApplicationList).ListMeta}
	for _, item := range obj.(*v1alpha1.InstrumentedApplicationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested instrumentedApplications.
func (c *FakeInstrumentedApplications) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(instrumentedapplicationsResource, c.ns, opts))

}

// Create takes the representation of a instrumentedApplication and creates it.  Returns the server's representation of the instrumentedApplication, and an error, if there is any.
func (c *FakeInstrumentedApplications) Create(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.CreateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(instrumentedapplicationsResource, c.ns, instrumentedApplication, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// Update takes the representation of a instrumentedApplication and updates it. Returns the server's representation of the instrumentedApplication, and an error, if there is any.
func (c *FakeInstrumentedApplications) Update(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(instrumentedapplicationsResource, c.ns, instrumentedApplication, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeInstrumentedApplications) UpdateStatus(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(instrumentedapplicationsResource, "status", c.ns, instrumentedApplication, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// Delete takes name of the instrumentedApplication and deletes it. Returns an error if one occurs.
func (c *FakeInstrumentedApplications) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(instrumentedapplicationsResource, c.ns, name, opts), &v1alpha1.InstrumentedApplication{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeInstrumentedApplications) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(instrumentedapplicationsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.InstrumentedApplicationList{})
	return err
}

// Patch applies the patch and returns the patched instrumentedApplication.
func (c *FakeInstrumentedApplications) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.InstrumentedApplication, err error) {
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(instrumentedapplicationsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied instrumentedApplication.
func (c *FakeInstrumentedApplications) Apply(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	if instrumentedApplication == nil {
		return nil, fmt.Errorf("instrumentedApplication provided to Apply must not be nil")
	}
	data, err := json.Marshal(instrumentedApplication)
	if err != nil {
		return nil, err
	}
	name := instrumentedApplication.Name
	if name == nil {
		return nil, fmt.Errorf("instrumentedApplication.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(instrumentedapplicationsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeInstrumentedApplications) ApplyStatus(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	if instrumentedApplication == nil {
		return nil, fmt.Errorf("instrumentedApplication provided to Apply must not be nil")
	}
	data, err := json.Marshal(instrumentedApplication)
	if err != nil {
		return nil, err
	}
	name := instrumentedApplication.Name
	if name == nil {
		return nil, fmt.Errorf("instrumentedApplication.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.InstrumentedApplication{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(instrumentedapplicationsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.InstrumentedApplication), err
}
