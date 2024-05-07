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

// InstrumentedApplicationsGetter has a method to return a InstrumentedApplicationInterface.
// A group's client should implement this interface.
type InstrumentedApplicationsGetter interface {
	InstrumentedApplications(namespace string) InstrumentedApplicationInterface
}

// InstrumentedApplicationInterface has methods to work with InstrumentedApplication resources.
type InstrumentedApplicationInterface interface {
	Create(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.CreateOptions) (*v1alpha1.InstrumentedApplication, error)
	Update(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (*v1alpha1.InstrumentedApplication, error)
	UpdateStatus(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (*v1alpha1.InstrumentedApplication, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.InstrumentedApplication, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.InstrumentedApplicationList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.InstrumentedApplication, err error)
	Apply(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error)
	ApplyStatus(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error)
	InstrumentedApplicationExpansion
}

// instrumentedApplications implements InstrumentedApplicationInterface
type instrumentedApplications struct {
	client rest.Interface
	ns     string
}

// newInstrumentedApplications returns a InstrumentedApplications
func newInstrumentedApplications(c *OdigosV1alpha1Client, namespace string) *instrumentedApplications {
	return &instrumentedApplications{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the instrumentedApplication, and returns the corresponding instrumentedApplication object, and an error if there is any.
func (c *instrumentedApplications) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of InstrumentedApplications that match those selectors.
func (c *instrumentedApplications) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.InstrumentedApplicationList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.InstrumentedApplicationList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested instrumentedApplications.
func (c *instrumentedApplications) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a instrumentedApplication and creates it.  Returns the server's representation of the instrumentedApplication, and an error, if there is any.
func (c *instrumentedApplications) Create(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.CreateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(instrumentedApplication).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a instrumentedApplication and updates it. Returns the server's representation of the instrumentedApplication, and an error, if there is any.
func (c *instrumentedApplications) Update(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(instrumentedApplication.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(instrumentedApplication).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *instrumentedApplications) UpdateStatus(ctx context.Context, instrumentedApplication *v1alpha1.InstrumentedApplication, opts v1.UpdateOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(instrumentedApplication.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(instrumentedApplication).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the instrumentedApplication and deletes it. Returns an error if one occurs.
func (c *instrumentedApplications) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *instrumentedApplications) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("instrumentedapplications").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched instrumentedApplication.
func (c *instrumentedApplications) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.InstrumentedApplication, err error) {
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied instrumentedApplication.
func (c *instrumentedApplications) Apply(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	if instrumentedApplication == nil {
		return nil, fmt.Errorf("instrumentedApplication provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(instrumentedApplication)
	if err != nil {
		return nil, err
	}
	name := instrumentedApplication.Name
	if name == nil {
		return nil, fmt.Errorf("instrumentedApplication.Name must be provided to Apply")
	}
	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *instrumentedApplications) ApplyStatus(ctx context.Context, instrumentedApplication *odigosv1alpha1.InstrumentedApplicationApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.InstrumentedApplication, err error) {
	if instrumentedApplication == nil {
		return nil, fmt.Errorf("instrumentedApplication provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(instrumentedApplication)
	if err != nil {
		return nil, err
	}

	name := instrumentedApplication.Name
	if name == nil {
		return nil, fmt.Errorf("instrumentedApplication.Name must be provided to Apply")
	}

	result = &v1alpha1.InstrumentedApplication{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("instrumentedapplications").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
