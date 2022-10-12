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

	v1alpha1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDestinations implements DestinationInterface
type FakeDestinations struct {
	Fake *FakeOdigosV1alpha1
	ns   string
}

var destinationsResource = schema.GroupVersionResource{Group: "odigos.io", Version: "v1alpha1", Resource: "destinations"}

var destinationsKind = schema.GroupVersionKind{Group: "odigos.io", Version: "v1alpha1", Kind: "Destination"}

// Get takes name of the destination, and returns the corresponding destination object, and an error if there is any.
func (c *FakeDestinations) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Destination, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(destinationsResource, c.ns, name), &v1alpha1.Destination{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Destination), err
}

// List takes label and field selectors, and returns the list of Destinations that match those selectors.
func (c *FakeDestinations) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.DestinationList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(destinationsResource, destinationsKind, c.ns, opts), &v1alpha1.DestinationList{})

	if obj == nil {
		return nil, err
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
		InvokesWatch(testing.NewWatchAction(destinationsResource, c.ns, opts))

}

// Create takes the representation of a destination and creates it.  Returns the server's representation of the destination, and an error, if there is any.
func (c *FakeDestinations) Create(ctx context.Context, destination *v1alpha1.Destination, opts v1.CreateOptions) (result *v1alpha1.Destination, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(destinationsResource, c.ns, destination), &v1alpha1.Destination{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Destination), err
}

// Update takes the representation of a destination and updates it. Returns the server's representation of the destination, and an error, if there is any.
func (c *FakeDestinations) Update(ctx context.Context, destination *v1alpha1.Destination, opts v1.UpdateOptions) (result *v1alpha1.Destination, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(destinationsResource, c.ns, destination), &v1alpha1.Destination{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Destination), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeDestinations) UpdateStatus(ctx context.Context, destination *v1alpha1.Destination, opts v1.UpdateOptions) (*v1alpha1.Destination, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(destinationsResource, "status", c.ns, destination), &v1alpha1.Destination{})

	if obj == nil {
		return nil, err
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
	action := testing.NewDeleteCollectionAction(destinationsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.DestinationList{})
	return err
}

// Patch applies the patch and returns the patched destination.
func (c *FakeDestinations) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Destination, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(destinationsResource, c.ns, name, pt, data, subresources...), &v1alpha1.Destination{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Destination), err
}
