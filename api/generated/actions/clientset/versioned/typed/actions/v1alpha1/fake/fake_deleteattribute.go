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

	v1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/actions/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeDeleteAttributes implements DeleteAttributeInterface
type FakeDeleteAttributes struct {
	Fake *FakeActionsV1alpha1
	ns   string
}

var deleteattributesResource = v1alpha1.SchemeGroupVersion.WithResource("deleteattributes")

var deleteattributesKind = v1alpha1.SchemeGroupVersion.WithKind("DeleteAttribute")

// Get takes name of the deleteAttribute, and returns the corresponding deleteAttribute object, and an error if there is any.
func (c *FakeDeleteAttributes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.DeleteAttribute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(deleteattributesResource, c.ns, name), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// List takes label and field selectors, and returns the list of DeleteAttributes that match those selectors.
func (c *FakeDeleteAttributes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.DeleteAttributeList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(deleteattributesResource, deleteattributesKind, c.ns, opts), &v1alpha1.DeleteAttributeList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.DeleteAttributeList{ListMeta: obj.(*v1alpha1.DeleteAttributeList).ListMeta}
	for _, item := range obj.(*v1alpha1.DeleteAttributeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested deleteAttributes.
func (c *FakeDeleteAttributes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(deleteattributesResource, c.ns, opts))

}

// Create takes the representation of a deleteAttribute and creates it.  Returns the server's representation of the deleteAttribute, and an error, if there is any.
func (c *FakeDeleteAttributes) Create(ctx context.Context, deleteAttribute *v1alpha1.DeleteAttribute, opts v1.CreateOptions) (result *v1alpha1.DeleteAttribute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(deleteattributesResource, c.ns, deleteAttribute), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// Update takes the representation of a deleteAttribute and updates it. Returns the server's representation of the deleteAttribute, and an error, if there is any.
func (c *FakeDeleteAttributes) Update(ctx context.Context, deleteAttribute *v1alpha1.DeleteAttribute, opts v1.UpdateOptions) (result *v1alpha1.DeleteAttribute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(deleteattributesResource, c.ns, deleteAttribute), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeDeleteAttributes) UpdateStatus(ctx context.Context, deleteAttribute *v1alpha1.DeleteAttribute, opts v1.UpdateOptions) (*v1alpha1.DeleteAttribute, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(deleteattributesResource, "status", c.ns, deleteAttribute), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// Delete takes name of the deleteAttribute and deletes it. Returns an error if one occurs.
func (c *FakeDeleteAttributes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(deleteattributesResource, c.ns, name, opts), &v1alpha1.DeleteAttribute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeDeleteAttributes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(deleteattributesResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.DeleteAttributeList{})
	return err
}

// Patch applies the patch and returns the patched deleteAttribute.
func (c *FakeDeleteAttributes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.DeleteAttribute, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(deleteattributesResource, c.ns, name, pt, data, subresources...), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied deleteAttribute.
func (c *FakeDeleteAttributes) Apply(ctx context.Context, deleteAttribute *actionsv1alpha1.DeleteAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.DeleteAttribute, err error) {
	if deleteAttribute == nil {
		return nil, fmt.Errorf("deleteAttribute provided to Apply must not be nil")
	}
	data, err := json.Marshal(deleteAttribute)
	if err != nil {
		return nil, err
	}
	name := deleteAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("deleteAttribute.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(deleteattributesResource, c.ns, *name, types.ApplyPatchType, data), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeDeleteAttributes) ApplyStatus(ctx context.Context, deleteAttribute *actionsv1alpha1.DeleteAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.DeleteAttribute, err error) {
	if deleteAttribute == nil {
		return nil, fmt.Errorf("deleteAttribute provided to Apply must not be nil")
	}
	data, err := json.Marshal(deleteAttribute)
	if err != nil {
		return nil, err
	}
	name := deleteAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("deleteAttribute.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(deleteattributesResource, c.ns, *name, types.ApplyPatchType, data, "status"), &v1alpha1.DeleteAttribute{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.DeleteAttribute), err
}
