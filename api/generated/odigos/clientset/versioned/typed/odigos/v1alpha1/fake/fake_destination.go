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

// FakeDestinations implements DestinationInterface
type FakeDestinations struct {
	Fake *FakeOdigosV1alpha1
	ns   string
}

var destinationsResource = v1alpha1.SchemeGroupVersion.WithResource("destinations")

var destinationsKind = v1alpha1.SchemeGroupVersion.WithKind("Destination")

// Get takes name of the destination, and returns the corresponding destination object, and an error if there is any.
func (c *FakeDestinations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Destination, err error) {
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(destinationsResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// List takes label and field selectors, and returns the list of Destinations that match those selectors.
func (c *FakeDestinations) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.DestinationList, err error) {
	emptyResult := &v1alpha1.DestinationList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(destinationsResource, destinationsKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.DestinationList{ListMeta: obj.(*v1alpha1.DestinationList).ListMeta}
	for _, item := range obj.(*v1alpha1.DestinationList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested destinations.
func (c *FakeDestinations) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(destinationsResource, c.ns, opts))

}

// Create takes the representation of a destination and creates it.  Returns the server's representation of the destination, and an error, if there is any.
func (c *FakeDestinations) Create(ctx context.Context, destination *v1alpha1.Destination, opts v1.CreateOptions) (result *v1alpha1.Destination, err error) {
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(destinationsResource, c.ns, destination, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// Update takes the representation of a destination and updates it. Returns the server's representation of the destination, and an error, if there is any.
func (c *FakeDestinations) Update(ctx context.Context, destination *v1alpha1.Destination, opts v1.UpdateOptions) (result *v1alpha1.Destination, err error) {
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(destinationsResource, c.ns, destination, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeDestinations) UpdateStatus(ctx context.Context, destination *v1alpha1.Destination, opts v1.UpdateOptions) (result *v1alpha1.Destination, err error) {
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(destinationsResource, "status", c.ns, destination, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// Delete takes name of the destination and deletes it. Returns an error if one occurs.
func (c *FakeDestinations) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(destinationsResource, c.ns, name, opts), &v1alpha1.Destination{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDestinations) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(destinationsResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.DestinationList{})
	return err
}

// Patch applies the patch and returns the patched destination.
func (c *FakeDestinations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Destination, err error) {
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(destinationsResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied destination.
func (c *FakeDestinations) Apply(ctx context.Context, destination *odigosv1alpha1.DestinationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Destination, err error) {
	if destination == nil {
		return nil, fmt.Errorf("destination provided to Apply must not be nil")
	}
	data, err := json.Marshal(destination)
	if err != nil {
		return nil, err
	}
	name := destination.Name
	if name == nil {
		return nil, fmt.Errorf("destination.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(destinationsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeDestinations) ApplyStatus(ctx context.Context, destination *odigosv1alpha1.DestinationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.Destination, err error) {
	if destination == nil {
		return nil, fmt.Errorf("destination provided to Apply must not be nil")
	}
	data, err := json.Marshal(destination)
	if err != nil {
		return nil, err
	}
	name := destination.Name
	if name == nil {
		return nil, fmt.Errorf("destination.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.Destination{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(destinationsResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.Destination), err
}