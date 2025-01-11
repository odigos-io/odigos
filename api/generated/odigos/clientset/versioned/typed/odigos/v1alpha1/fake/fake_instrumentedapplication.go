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

// fakeInstrumentedApplications implements InstrumentedApplicationInterface
type fakeInstrumentedApplications struct {
	*gentype.FakeClientWithListAndApply[*v1alpha1.InstrumentedApplication, *v1alpha1.InstrumentedApplicationList, *odigosv1alpha1.InstrumentedApplicationApplyConfiguration]
	Fake *FakeOdigosV1alpha1
}

func newFakeInstrumentedApplications(fake *FakeOdigosV1alpha1, namespace string) typedodigosv1alpha1.InstrumentedApplicationInterface {
	return &fakeInstrumentedApplications{
		gentype.NewFakeClientWithListAndApply[*v1alpha1.InstrumentedApplication, *v1alpha1.InstrumentedApplicationList, *odigosv1alpha1.InstrumentedApplicationApplyConfiguration](
			fake.Fake,
			namespace,
			v1alpha1.SchemeGroupVersion.WithResource("instrumentedapplications"),
			v1alpha1.SchemeGroupVersion.WithKind("InstrumentedApplication"),
			func() *v1alpha1.InstrumentedApplication { return &v1alpha1.InstrumentedApplication{} },
			func() *v1alpha1.InstrumentedApplicationList { return &v1alpha1.InstrumentedApplicationList{} },
			func(dst, src *v1alpha1.InstrumentedApplicationList) { dst.ListMeta = src.ListMeta },
			func(list *v1alpha1.InstrumentedApplicationList) []*v1alpha1.InstrumentedApplication {
				return gentype.ToPointerSlice(list.Items)
			},
			func(list *v1alpha1.InstrumentedApplicationList, items []*v1alpha1.InstrumentedApplication) {
				list.Items = gentype.FromPointerSlice(items)
			},
		),
		fake,
	}
}
