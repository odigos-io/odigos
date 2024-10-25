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

// FakeRenameAttributes implements RenameAttributeInterface
type FakeRenameAttributes struct {
	Fake *FakeActionsV1alpha1
	ns   string
}

var renameattributesResource = v1alpha1.SchemeGroupVersion.WithResource("renameattributes")

var renameattributesKind = v1alpha1.SchemeGroupVersion.WithKind("RenameAttribute")

// Get takes name of the renameAttribute, and returns the corresponding renameAttribute object, and an error if there is any.
func (c *FakeRenameAttributes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.RenameAttribute, err error) {
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewGetActionWithOptions(renameattributesResource, c.ns, name, options), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// List takes label and field selectors, and returns the list of RenameAttributes that match those selectors.
func (c *FakeRenameAttributes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RenameAttributeList, err error) {
	emptyResult := &v1alpha1.RenameAttributeList{}
	obj, err := c.Fake.
		Invokes(testing.NewListActionWithOptions(renameattributesResource, renameattributesKind, c.ns, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.RenameAttributeList{ListMeta: obj.(*v1alpha1.RenameAttributeList).ListMeta}
	for _, item := range obj.(*v1alpha1.RenameAttributeList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested renameAttributes.
func (c *FakeRenameAttributes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchActionWithOptions(renameattributesResource, c.ns, opts))

}

// Create takes the representation of a renameAttribute and creates it.  Returns the server's representation of the renameAttribute, and an error, if there is any.
func (c *FakeRenameAttributes) Create(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.CreateOptions) (result *v1alpha1.RenameAttribute, err error) {
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewCreateActionWithOptions(renameattributesResource, c.ns, renameAttribute, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// Update takes the representation of a renameAttribute and updates it. Returns the server's representation of the renameAttribute, and an error, if there is any.
func (c *FakeRenameAttributes) Update(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (result *v1alpha1.RenameAttribute, err error) {
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateActionWithOptions(renameattributesResource, c.ns, renameAttribute, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeRenameAttributes) UpdateStatus(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (result *v1alpha1.RenameAttribute, err error) {
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceActionWithOptions(renameattributesResource, "status", c.ns, renameAttribute, opts), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// Delete takes name of the renameAttribute and deletes it. Returns an error if one occurs.
func (c *FakeRenameAttributes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(renameattributesResource, c.ns, name, opts), &v1alpha1.RenameAttribute{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeRenameAttributes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionActionWithOptions(renameattributesResource, c.ns, opts, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.RenameAttributeList{})
	return err
}

// Patch applies the patch and returns the patched renameAttribute.
func (c *FakeRenameAttributes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.RenameAttribute, err error) {
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(renameattributesResource, c.ns, name, pt, data, opts, subresources...), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied renameAttribute.
func (c *FakeRenameAttributes) Apply(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error) {
	if renameAttribute == nil {
		return nil, fmt.Errorf("renameAttribute provided to Apply must not be nil")
	}
	data, err := json.Marshal(renameAttribute)
	if err != nil {
		return nil, err
	}
	name := renameAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("renameAttribute.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(renameattributesResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions()), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeRenameAttributes) ApplyStatus(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error) {
	if renameAttribute == nil {
		return nil, fmt.Errorf("renameAttribute provided to Apply must not be nil")
	}
	data, err := json.Marshal(renameAttribute)
	if err != nil {
		return nil, err
	}
	name := renameAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("renameAttribute.Name must be provided to Apply")
	}
	emptyResult := &v1alpha1.RenameAttribute{}
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceActionWithOptions(renameattributesResource, c.ns, *name, types.ApplyPatchType, data, opts.ToPatchOptions(), "status"), emptyResult)

	if obj == nil {
		return emptyResult, err
	}
	return obj.(*v1alpha1.RenameAttribute), err
}