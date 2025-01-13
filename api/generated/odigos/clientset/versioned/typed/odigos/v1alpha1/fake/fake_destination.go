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
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	typedodigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	gentype "k8s.io/client-go/gentype"
)

// fakeDestinations implements DestinationInterface
type fakeDestinations struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.Destination, *v1alpha1.DestinationList, *odigosv1alpha1.DestinationApplyConfiguration]
	Fake *FakeOdigosV1alpha1
}

func newFakeDestinations(fake *FakeOdigosV1alpha1, namespace string) typedodigosv1alpha1.DestinationInterface {
	return &fakeDestinations{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.Destination, *v1alpha1.DestinationList, *odigosv1alpha1.DestinationApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("destinations"),
			v1alpha1.SchemeGroupVersion.WithKind("Destination"),
			func() *v1alpha1.Destination { return &v1alpha1.Destination{} },
			func() *v1alpha1.DestinationList { return &v1alpha1.DestinationList{} },
			func(dst, src *v1alpha1.DestinationList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.DestinationList) []*v1alpha1.Destination {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.DestinationList, items []*v1alpha1.Destination) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
