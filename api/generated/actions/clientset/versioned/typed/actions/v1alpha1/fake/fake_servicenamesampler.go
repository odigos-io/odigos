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
	v1alpha1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/applyconfiguration/actions/v1alpha1"
	typedactionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/typed/actions/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeServiceNameSamplers implements ServiceNameSamplerInterface
type fakeServiceNameSamplers struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.ServiceNameSampler, *v1alpha1.ServiceNameSamplerList, *actionsv1alpha1.ServiceNameSamplerApplyConfiguration]
	Fake *FakeActionsV1alpha1
}

func newFakeServiceNameSamplers(fake *FakeActionsV1alpha1, namespace string) typedactionsv1alpha1.ServiceNameSamplerInterface {
	return &fakeServiceNameSamplers{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.ServiceNameSampler, *v1alpha1.ServiceNameSamplerList, *actionsv1alpha1.ServiceNameSamplerApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("servicenamesamplers"),
			v1alpha1.SchemeGroupVersion.WithKind("ServiceNameSampler"),
			func() *v1alpha1.ServiceNameSampler { return &v1alpha1.ServiceNameSampler{} },
			func() *v1alpha1.ServiceNameSamplerList { return &v1alpha1.ServiceNameSamplerList{} },
			func(dst, src *v1alpha1.ServiceNameSamplerList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.ServiceNameSamplerList) []*v1alpha1.ServiceNameSampler {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.ServiceNameSamplerList, items []*v1alpha1.ServiceNameSampler) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
