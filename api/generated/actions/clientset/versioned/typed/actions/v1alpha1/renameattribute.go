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
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/actions/v1alpha1"
	scheme "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// RenameAttributesGetter has a method to return a RenameAttributeInterface.
// A group's client should implement this interface.
type RenameAttributesGetter interface {
	RenameAttributes(namespace string) RenameAttributeInterface
}

// RenameAttributeInterface has methods to work with RenameAttribute resources.
type RenameAttributeInterface interface {
	Create(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.CreateOptions) (*v1alpha1.RenameAttribute, error)
	Update(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (*v1alpha1.RenameAttribute, error)
	UpdateStatus(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (*v1alpha1.RenameAttribute, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.RenameAttribute, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.RenameAttributeList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.RenameAttribute, err error)
	Apply(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error)
	ApplyStatus(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error)
	RenameAttributeExpansion
}

// renameAttributes implements RenameAttributeInterface
type renameAttributes struct {
	client rest.Interface
	ns     string
}

// newRenameAttributes returns a RenameAttributes
func newRenameAttributes(c *ActionsV1alpha1Client, namespace string) *renameAttributes {
	return &renameAttributes{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the renameAttribute, and returns the corresponding renameAttribute object, and an error if there is any.
func (c *renameAttributes) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.RenameAttribute, err error) {
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("renameattributes").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of RenameAttributes that match those selectors.
func (c *renameAttributes) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.RenameAttributeList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.RenameAttributeList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("renameattributes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested renameAttributes.
func (c *renameAttributes) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("renameattributes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a renameAttribute and creates it.  Returns the server's representation of the renameAttribute, and an error, if there is any.
func (c *renameAttributes) Create(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.CreateOptions) (result *v1alpha1.RenameAttribute, err error) {
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("renameattributes").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(renameAttribute).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a renameAttribute and updates it. Returns the server's representation of the renameAttribute, and an error, if there is any.
func (c *renameAttributes) Update(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (result *v1alpha1.RenameAttribute, err error) {
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("renameattributes").
		Name(renameAttribute.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(renameAttribute).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *renameAttributes) UpdateStatus(ctx context.Context, renameAttribute *v1alpha1.RenameAttribute, opts v1.UpdateOptions) (result *v1alpha1.RenameAttribute, err error) {
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("renameattributes").
		Name(renameAttribute.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(renameAttribute).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the renameAttribute and deletes it. Returns an error if one occurs.
func (c *renameAttributes) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("renameattributes").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *renameAttributes) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("renameattributes").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched renameAttribute.
func (c *renameAttributes) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.RenameAttribute, err error) {
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("renameattributes").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied renameAttribute.
func (c *renameAttributes) Apply(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error) {
	if renameAttribute == nil {
		return nil, fmt.Errorf("renameAttribute provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(renameAttribute)
	if err != nil {
		return nil, err
	}
	name := renameAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("renameAttribute.Name must be provided to Apply")
	}
	result = &v1alpha1.RenameAttribute{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("renameattributes").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *renameAttributes) ApplyStatus(ctx context.Context, renameAttribute *actionsv1alpha1.RenameAttributeApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.RenameAttribute, err error) {
	if renameAttribute == nil {
		return nil, fmt.Errorf("renameAttribute provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(renameAttribute)
	if err != nil {
		return nil, err
	}

	name := renameAttribute.Name
	if name == nil {
		return nil, fmt.Errorf("renameAttribute.Name must be provided to Apply")
	}

	result = &v1alpha1.RenameAttribute{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("renameattributes").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
