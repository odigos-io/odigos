/*
Copyright 2024.

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

package v1alpha1

import (
	"context"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DataStream configures a group/stream/sub-pipeline to export telemetry data from explicit sources to explicit destinations.
// At the moment this is used only for naming the stream.
// In the future it will be used to configure settings/features of the stream.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels=odigos.io/config=1
// +kubebuilder:metadata:labels=odigos.io/system-object=true
type DataStream struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

// +kubebuilder:object:root=true
type DataStreamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataStream `json:"items"`
}

// +kubebuilder:object:generate=false
func init() {
	SchemeBuilder.Register(&DataStream{}, &DataStreamList{})
}

func GetDataStream(ctx context.Context, kubeClient client.Client, streamName string) (*DataStream, error) {
	stream := &DataStream{}
	objectKey := client.ObjectKey{
		Name:      streamName,
		Namespace: env.GetCurrentNamespace(),
	}

	err := kubeClient.Get(ctx, objectKey, stream)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	return stream, nil
}

func CreateDataStream(ctx context.Context, kubeClient client.Client, streamName string) (*DataStream, error) {
	stream, err := GetDataStream(ctx, kubeClient, streamName)
	if err != nil {
		return nil, err
	}
	if stream != nil {
		return stream, nil
	}

	stream = &DataStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      streamName,
			Namespace: env.GetCurrentNamespace(),
		},
	}

	err = kubeClient.Create(ctx, stream)
	if err != nil {
		return nil, err
	}

	return stream, nil
}
