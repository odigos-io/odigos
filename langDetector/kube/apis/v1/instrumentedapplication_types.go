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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstrumentedApplicationSpec defines the desired state of InstrumentedApplication
type InstrumentedApplicationSpec struct {
	Ref ApplicationReference `json:"ref"`

	// +optional
	Languages     []LanguageByContainer `json:"languages,omitempty"`
	Instrumented  bool                  `json:"instrumented"`
	CollectorAddr string                `json:"collectorAddr,omitempty"`
}

type LanguageByContainer struct {
	ContainerName string              `json:"containerName"`
	Language      ProgrammingLanguage `json:"language"`
	ProcessName   string              `json:"processName,omitempty"`
}

//+kubebuilder:validation:Enum=java;python;go;dotnet;javascript
type ProgrammingLanguage string

const (
	JavaProgrammingLanguage       ProgrammingLanguage = "java"
	PythonProgrammingLanguage     ProgrammingLanguage = "python"
	GoProgrammingLanguage         ProgrammingLanguage = "go"
	DotNetProgrammingLanguage     ProgrammingLanguage = "dotnet"
	JavascriptProgrammingLanguage ProgrammingLanguage = "javascript"
)

//+kubebuilder:validation:Enum=deployment;statefulset
type ApplicationType string

const (
	DeploymentApplicationType  ApplicationType = "deployment"
	StatefulSetApplicationType ApplicationType = "statefulset"
)

type ApplicationReference struct {
	Type      ApplicationType `json:"type"`
	Namespace string          `json:"namespace"`
	Name      string          `json:"name"`
}

// InstrumentedApplicationStatus defines the observed state of InstrumentedApplication
type InstrumentedApplicationStatus struct {
	LangDetection LangDetectionStatus `json:"langDetection,omitempty"`
}

type LangDetectionStatus struct {
	Phase LangDetectionPhase `json:"phase,omitempty"`
}

//+kubebuilder:validation:Enum=Pending;Running;Completed;Error
type LangDetectionPhase string

const (
	PendingLangDetectionPhase   LangDetectionPhase = "Pending"
	RunningLangDetectionPhase   LangDetectionPhase = "Running"
	CompletedLangDetectionPhase LangDetectionPhase = "Completed"
	ErrorLangDetectionPhase     LangDetectionPhase = "Error"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InstrumentedApplication is the Schema for the instrumentedapplications API
type InstrumentedApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentedApplicationSpec   `json:"spec,omitempty"`
	Status InstrumentedApplicationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstrumentedApplicationList contains a list of InstrumentedApplication
type InstrumentedApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentedApplication `json:"items"`
}
