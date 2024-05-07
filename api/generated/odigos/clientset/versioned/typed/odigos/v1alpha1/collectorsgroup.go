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

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	scheme "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/scheme"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// CollectorsGroupsGetter has a method to return a CollectorsGroupInterface.
// A group's client should implement this interface.
type CollectorsGroupsGetter interface {
	CollectorsGroups(namespace string) CollectorsGroupInterface
}

// CollectorsGroupInterface has methods to work with CollectorsGroup resources.
type CollectorsGroupInterface interface {
	Create(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.CreateOptions) (*v1alpha1.CollectorsGroup, error)
	Update(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.UpdateOptions) (*v1alpha1.CollectorsGroup, error)
	UpdateStatus(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.UpdateOptions) (*v1alpha1.CollectorsGroup, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.CollectorsGroup, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.CollectorsGroupList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CollectorsGroup, err error)
	Apply(ctx context.Context, collectorsGroup *odigosv1alpha1.CollectorsGroupApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.CollectorsGroup, err error)
	ApplyStatus(ctx context.Context, collectorsGroup *odigosv1alpha1.CollectorsGroupApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.CollectorsGroup, err error)
	CollectorsGroupExpansion
}

// collectorsGroups implements CollectorsGroupInterface
type collectorsGroups struct {
	client rest.Interface
	ns     string
}

// newCollectorsGroups returns a CollectorsGroups
func newCollectorsGroups(c *OdigosV1alpha1Client, namespace string) *collectorsGroups {
	return &collectorsGroups{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the collectorsGroup, and returns the corresponding collectorsGroup object, and an error if there is any.
func (c *collectorsGroups) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.CollectorsGroup, err error) {
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of CollectorsGroups that match those selectors.
func (c *collectorsGroups) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.CollectorsGroupList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.CollectorsGroupList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("collectorsgroups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested collectorsGroups.
func (c *collectorsGroups) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("collectorsgroups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a collectorsGroup and creates it.  Returns the server's representation of the collectorsGroup, and an error, if there is any.
func (c *collectorsGroups) Create(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.CreateOptions) (result *v1alpha1.CollectorsGroup, err error) {
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("collectorsgroups").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(collectorsGroup).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a collectorsGroup and updates it. Returns the server's representation of the collectorsGroup, and an error, if there is any.
func (c *collectorsGroups) Update(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.UpdateOptions) (result *v1alpha1.CollectorsGroup, err error) {
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(collectorsGroup.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(collectorsGroup).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *collectorsGroups) UpdateStatus(ctx context.Context, collectorsGroup *v1alpha1.CollectorsGroup, opts v1.UpdateOptions) (result *v1alpha1.CollectorsGroup, err error) {
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(collectorsGroup.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(collectorsGroup).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the collectorsGroup and deletes it. Returns an error if one occurs.
func (c *collectorsGroups) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *collectorsGroups) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("collectorsgroups").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched collectorsGroup.
func (c *collectorsGroups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CollectorsGroup, err error) {
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied collectorsGroup.
func (c *collectorsGroups) Apply(ctx context.Context, collectorsGroup *odigosv1alpha1.CollectorsGroupApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.CollectorsGroup, err error) {
	if collectorsGroup == nil {
		return nil, fmt.Errorf("collectorsGroup provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(collectorsGroup)
	if err != nil {
		return nil, err
	}
	name := collectorsGroup.Name
	if name == nil {
		return nil, fmt.Errorf("collectorsGroup.Name must be provided to Apply")
	}
	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *collectorsGroups) ApplyStatus(ctx context.Context, collectorsGroup *odigosv1alpha1.CollectorsGroupApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.CollectorsGroup, err error) {
	if collectorsGroup == nil {
		return nil, fmt.Errorf("collectorsGroup provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(collectorsGroup)
	if err != nil {
		return nil, err
	}

	name := collectorsGroup.Name
	if name == nil {
		return nil, fmt.Errorf("collectorsGroup.Name must be provided to Apply")
	}

	result = &v1alpha1.CollectorsGroup{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("collectorsgroups").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
