package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

const ActionSpanRenamer = "SpanRenamer"

// generic span renamer config that works for any programming language and scope name.
// can be used as fast response to rename high-cardinality spans if they are ever created.
type SpanRenamerGeneric struct {
	// the programming language which the renamed spans are written in.
	ProgrammingLanguage common.ProgrammingLanguage `json:"programmingLanguage"`

	// the name of the opentelemetry intrumentation scope which the renamed spans are written in.
	ScopeName string `json:"scopeName"`

	// if set, spans matching the above conditions will be renamed to this constant value.
	ConstantSpanName string `json:"constantSpanName"`
}

// can be used to rename java quarts spans.
// by default the span name will be set to `{JobGroup}.{JobName}`.
// these are typically low cardinality values, but not always.
// the job group and name can be set to high cardinality values in code,
// in which case they need to be templated to a low cardinality value replacement.
type SpanRenamerJavaQuartz struct {

	// if set, the job group will be replaced in span names to this templated value.
	JobGroupTemplate string `json:"jobGroupTemplate,omitempty"`

	// if set, the job name will be replaced in span names to this templated value.
	JobNameTemplate string `json:"jobNameTemplate,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type SpanRenamerConfig struct {

	// generic span renamer config that works for any programming language and scope name.
	// can be used as fast response to rename high-cardinality spans if they are ever created.
	Generic *SpanRenamerGeneric `json:"generic,omitempty"`

	// java quarts span renamer config that can be used to rename java quarts spans.
	JavaQuartz *SpanRenamerJavaQuartz `json:"javaQuartz,omitempty"`
}

func (SpanRenamerConfig) ProcessorType() string {
	return "odigosspanrenamer"
}

func (SpanRenamerConfig) OrderHint() int {
	return 1
}

func (SpanRenamerConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
