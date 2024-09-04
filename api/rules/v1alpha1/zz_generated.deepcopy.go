//go:build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DbStatementPayloadCollectionRule) DeepCopyInto(out *DbStatementPayloadCollectionRule) {
	*out = *in
	if in.MaxPayloadLength != nil {
		in, out := &in.MaxPayloadLength, &out.MaxPayloadLength
		*out = new(int64)
		**out = **in
	}
	if in.DropPartialPayloads != nil {
		in, out := &in.DropPartialPayloads, &out.DropPartialPayloads
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DbStatementPayloadCollectionRule.
func (in *DbStatementPayloadCollectionRule) DeepCopy() *DbStatementPayloadCollectionRule {
	if in == nil {
		return nil
	}
	out := new(DbStatementPayloadCollectionRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HttpPayloadCollectionRule) DeepCopyInto(out *HttpPayloadCollectionRule) {
	*out = *in
	if in.MimeTypes != nil {
		in, out := &in.MimeTypes, &out.MimeTypes
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
	if in.MaxPayloadLength != nil {
		in, out := &in.MaxPayloadLength, &out.MaxPayloadLength
		*out = new(int64)
		**out = **in
	}
	if in.DropPartialPayloads != nil {
		in, out := &in.DropPartialPayloads, &out.DropPartialPayloads
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HttpPayloadCollectionRule.
func (in *HttpPayloadCollectionRule) DeepCopy() *HttpPayloadCollectionRule {
	if in == nil {
		return nil
	}
	out := new(HttpPayloadCollectionRule)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *InstrumentationLibraryId) DeepCopyInto(out *InstrumentationLibraryId) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new InstrumentationLibraryId.
func (in *InstrumentationLibraryId) DeepCopy() *InstrumentationLibraryId {
	if in == nil {
		return nil
	}
	out := new(InstrumentationLibraryId)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PayloadCollection) DeepCopyInto(out *PayloadCollection) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PayloadCollection.
func (in *PayloadCollection) DeepCopy() *PayloadCollection {
	if in == nil {
		return nil
	}
	out := new(PayloadCollection)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PayloadCollection) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PayloadCollectionList) DeepCopyInto(out *PayloadCollectionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PayloadCollection, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PayloadCollectionList.
func (in *PayloadCollectionList) DeepCopy() *PayloadCollectionList {
	if in == nil {
		return nil
	}
	out := new(PayloadCollectionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PayloadCollectionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PayloadCollectionSpec) DeepCopyInto(out *PayloadCollectionSpec) {
	*out = *in
	if in.Workloads != nil {
		in, out := &in.Workloads, &out.Workloads
		*out = new([]workload.PodWorkload)
		if **in != nil {
			in, out := *in, *out
			*out = make([]workload.PodWorkload, len(*in))
			copy(*out, *in)
		}
	}
	if in.InstrumentationLibraries != nil {
		in, out := &in.InstrumentationLibraries, &out.InstrumentationLibraries
		*out = new([]InstrumentationLibraryId)
		if **in != nil {
			in, out := *in, *out
			*out = make([]InstrumentationLibraryId, len(*in))
			copy(*out, *in)
		}
	}
	if in.HttpRequest != nil {
		in, out := &in.HttpRequest, &out.HttpRequest
		*out = new(HttpPayloadCollectionRule)
		(*in).DeepCopyInto(*out)
	}
	if in.HttpResponse != nil {
		in, out := &in.HttpResponse, &out.HttpResponse
		*out = new(HttpPayloadCollectionRule)
		(*in).DeepCopyInto(*out)
	}
	if in.DbStatement != nil {
		in, out := &in.DbStatement, &out.DbStatement
		*out = new(DbStatementPayloadCollectionRule)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PayloadCollectionSpec.
func (in *PayloadCollectionSpec) DeepCopy() *PayloadCollectionSpec {
	if in == nil {
		return nil
	}
	out := new(PayloadCollectionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PayloadCollectionStatus) DeepCopyInto(out *PayloadCollectionStatus) {
	*out = *in
	if in.Conditions != nil {
		in, out := &in.Conditions, &out.Conditions
		*out = make([]v1.Condition, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PayloadCollectionStatus.
func (in *PayloadCollectionStatus) DeepCopy() *PayloadCollectionStatus {
	if in == nil {
		return nil
	}
	out := new(PayloadCollectionStatus)
	in.DeepCopyInto(out)
	return out
}
