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

// ProbabilisticSamplersGetter has a method to return a ProbabilisticSamplerInterface.
// A group's client should implement this interface.
type ProbabilisticSamplersGetter interface {
	ProbabilisticSamplers(namespace string) ProbabilisticSamplerInterface
}

// ProbabilisticSamplerInterface has methods to work with ProbabilisticSampler resources.
type ProbabilisticSamplerInterface interface {
	Create(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.CreateOptions) (*v1alpha1.ProbabilisticSampler, error)
	Update(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.UpdateOptions) (*v1alpha1.ProbabilisticSampler, error)
	UpdateStatus(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.UpdateOptions) (*v1alpha1.ProbabilisticSampler, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.ProbabilisticSampler, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ProbabilisticSamplerList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ProbabilisticSampler, err error)
	Apply(ctx context.Context, probabilisticSampler *actionsv1alpha1.ProbabilisticSamplerApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ProbabilisticSampler, err error)
	ApplyStatus(ctx context.Context, probabilisticSampler *actionsv1alpha1.ProbabilisticSamplerApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ProbabilisticSampler, err error)
	ProbabilisticSamplerExpansion
}

// probabilisticSamplers implements ProbabilisticSamplerInterface
type probabilisticSamplers struct {
	client rest.Interface
	ns     string
}

// newProbabilisticSamplers returns a ProbabilisticSamplers
func newProbabilisticSamplers(c *ActionsV1alpha1Client, namespace string) *probabilisticSamplers {
	return &probabilisticSamplers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the probabilisticSampler, and returns the corresponding probabilisticSampler object, and an error if there is any.
func (c *probabilisticSamplers) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of ProbabilisticSamplers that match those selectors.
func (c *probabilisticSamplers) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ProbabilisticSamplerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.ProbabilisticSamplerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested probabilisticSamplers.
func (c *probabilisticSamplers) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a probabilisticSampler and creates it.  Returns the server's representation of the probabilisticSampler, and an error, if there is any.
func (c *probabilisticSamplers) Create(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.CreateOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(probabilisticSampler).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a probabilisticSampler and updates it. Returns the server's representation of the probabilisticSampler, and an error, if there is any.
func (c *probabilisticSamplers) Update(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.UpdateOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(probabilisticSampler.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(probabilisticSampler).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *probabilisticSamplers) UpdateStatus(ctx context.Context, probabilisticSampler *v1alpha1.ProbabilisticSampler, opts v1.UpdateOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(probabilisticSampler.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(probabilisticSampler).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the probabilisticSampler and deletes it. Returns an error if one occurs.
func (c *probabilisticSamplers) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *probabilisticSamplers) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched probabilisticSampler.
func (c *probabilisticSamplers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ProbabilisticSampler, err error) {
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied probabilisticSampler.
func (c *probabilisticSamplers) Apply(ctx context.Context, probabilisticSampler *actionsv1alpha1.ProbabilisticSamplerApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	if probabilisticSampler == nil {
		return nil, fmt.Errorf("probabilisticSampler provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(probabilisticSampler)
	if err != nil {
		return nil, err
	}
	name := probabilisticSampler.Name
	if name == nil {
		return nil, fmt.Errorf("probabilisticSampler.Name must be provided to Apply")
	}
	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *probabilisticSamplers) ApplyStatus(ctx context.Context, probabilisticSampler *actionsv1alpha1.ProbabilisticSamplerApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.ProbabilisticSampler, err error) {
	if probabilisticSampler == nil {
		return nil, fmt.Errorf("probabilisticSampler provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(probabilisticSampler)
	if err != nil {
		return nil, err
	}

	name := probabilisticSampler.Name
	if name == nil {
		return nil, fmt.Errorf("probabilisticSampler.Name must be provided to Apply")
	}

	result = &v1alpha1.ProbabilisticSampler{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("probabilisticsamplers").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
