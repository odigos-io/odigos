package instrumentationrules

import (
	"errors"
	"fmt"
	"time"
)

// CustomProbeCategory classifies why a custom probe was added.
// +kubebuilder:validation:Enum=observability;security;performance
type CustomProbeCategory string

const (
	CustomProbeCategoryObservability CustomProbeCategory = "observability"
	CustomProbeCategorySecurity      CustomProbeCategory = "security"
	CustomProbeCategoryPerformance   CustomProbeCategory = "performance"
)

func (c CustomProbeCategory) Verify() error {
	switch c {
	case "", CustomProbeCategoryObservability, CustomProbeCategorySecurity, CustomProbeCategoryPerformance:
		return nil
	default:
		return fmt.Errorf("invalid category %q", c)
	}
}

func verifyAddedAt(addedAt string) error {
	if addedAt == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, addedAt); err != nil {
		return fmt.Errorf("addedAt must be an RFC3339 timestamp: %w", err)
	}
	return nil
}

// CustomProbeMetadata is bookkeeping stored with a probe on an InstrumentationRule.
// It is not copied onto the InstrumentationConfig sent to agents.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomProbeMetadata struct {
	// Notes is a free-form text field that allows you to attach notes regarding this probe for convenience.
	// For example: why it was added. Odigos does not use or assume any meaning from this field.
	Notes string `json:"notes,omitempty" yaml:"notes,omitempty"`
	// Category classifies why this probe was added.
	Category CustomProbeCategory `json:"category,omitempty" yaml:"category,omitempty"`
	// AddedAt is the time this probe was added, in RFC3339 form.
	// +kubebuilder:validation:Format=date-time
	AddedAt string `json:"addedAt,omitempty" yaml:"addedAt,omitempty"`
}

func (m CustomProbeMetadata) Verify() error {
	if err := m.Category.Verify(); err != nil {
		return err
	}
	return verifyAddedAt(m.AddedAt)
}

// CustomInstrumentations is the probe list sent to agents via InstrumentationConfig.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomInstrumentations struct {
	Golang []GolangCustomProbe `json:"golang,omitempty" yaml:"golang,omitempty"`
	Java   []JavaCustomProbe   `json:"java,omitempty" yaml:"java,omitempty"`
	Cpp    []CppCustomProbe    `json:"cpp,omitempty" yaml:"cpp,omitempty"`
	Php    []PhpCustomProbe    `json:"php,omitempty" yaml:"php,omitempty"`
}

// Verify iterates all custom instrumentations' probes and validates them.
// TODO: use generics and reflection to reduce boilerplate code.
func (ci *CustomInstrumentations) Verify() error {
	if ci == nil {
		return nil
	}
	// Validate Golang probes
	for _, p := range ci.Golang {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for golang custom instrumentation: %w", err)
		}
	}
	// Validate Java probes
	for _, p := range ci.Java {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for java custom instrumentation: %w", err)
		}
	}
	// Validate PHP probes
	for _, p := range ci.Php {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for php custom instrumentation: %w", err)
		}
	}
	return nil
}

// CustomInstrumentationsWithMetadata is the InstrumentationRule form of custom probes.
// Metadata stays on the rule and is dropped when probes are copied to InstrumentationConfig.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomInstrumentationsWithMetadata struct {
	Golang []GolangCustomProbeWithMetadata `json:"golang,omitempty" yaml:"golang,omitempty"`
	Java   []JavaCustomProbeWithMetadata   `json:"java,omitempty" yaml:"java,omitempty"`
	Cpp    []CppCustomProbeWithMetadata    `json:"cpp,omitempty" yaml:"cpp,omitempty"`
	Php    []PhpCustomProbeWithMetadata    `json:"php,omitempty" yaml:"php,omitempty"`
}

func (ci *CustomInstrumentationsWithMetadata) Verify() error {
	if ci == nil {
		return nil
	}
	for _, p := range ci.Golang {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for golang custom instrumentation: %w", err)
		}
	}
	for _, p := range ci.Java {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for java custom instrumentation: %w", err)
		}
	}
	for _, p := range ci.Php {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for php custom instrumentation: %w", err)
		}
	}
	for _, p := range ci.Cpp {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for cpp custom instrumentation: %w", err)
		}
	}
	return nil
}

// ToConfig returns the agent-facing probes with metadata removed.
func (ci *CustomInstrumentationsWithMetadata) ToConfig() *CustomInstrumentations {
	if ci == nil {
		return nil
	}
	out := &CustomInstrumentations{}
	if ci.Golang != nil {
		out.Golang = make([]GolangCustomProbe, len(ci.Golang))
		for i := range ci.Golang {
			out.Golang[i] = ci.Golang[i].GolangCustomProbe
		}
	}
	if ci.Java != nil {
		out.Java = make([]JavaCustomProbe, len(ci.Java))
		for i := range ci.Java {
			out.Java[i] = ci.Java[i].JavaCustomProbe
		}
	}
	if ci.Cpp != nil {
		out.Cpp = make([]CppCustomProbe, len(ci.Cpp))
		for i := range ci.Cpp {
			out.Cpp[i] = ci.Cpp[i].CppCustomProbe
		}
	}
	if ci.Php != nil {
		out.Php = make([]PhpCustomProbe, len(ci.Php))
		for i := range ci.Php {
			out.Php[i] = ci.Php[i].PhpCustomProbe
		}
	}
	return out
}

// java custom probe contains the details for a custom probe for java applications,
// which includes the class name and method name to be instrumented.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type JavaCustomProbe struct {
	ClassName  string `json:"className,omitempty" yaml:"className,omitempty"`
	MethodName string `json:"methodName,omitempty" yaml:"methodName,omitempty"`
}

// JavaCustomProbeWithMetadata is a Java probe plus rule-only metadata.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type JavaCustomProbeWithMetadata struct {
	JavaCustomProbe     `json:",inline"`
	CustomProbeMetadata `json:",inline"`
}

// For java we always require both class name and method name
func (jcp *JavaCustomProbe) Verify() error {
	if jcp.ClassName == "" {
		return errors.New("class name is required")
	}
	if jcp.MethodName == "" {
		return errors.New("method Name is required")
	}
	return nil
}

func (jcp *JavaCustomProbeWithMetadata) Verify() error {
	if err := jcp.CustomProbeMetadata.Verify(); err != nil {
		return err
	}
	return jcp.JavaCustomProbe.Verify()
}

// TODO(Barun): remove the String if you're not using it
// String implements fmt.Stringer for convenient logging/diagnostics.
func (jcp *JavaCustomProbe) String() string {
	return fmt.Sprintf("%s.%s", jcp.ClassName, jcp.MethodName)
}

// golang custom probe contains the details for a custom probe for golang applications,
// which includes the package name, function name or receiver name and method name to be instrumented.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type GolangCustomProbe struct {
	// PackageName is the name of the golang package (ie net/http); Package name is always required
	PackageName string `json:"packageName" yaml:"packageName"`
	// FunctionName is the name of the golang function to be instrumented, ie package name is "net/http" and the function
	// name will be "ListenAndServe"; Function name is disallowed if ReceiverName and ReceiverMethodName are provided
	FunctionName string `json:"functionName,omitempty" yaml:"functionName,omitempty"`
	// ReceiverName is the name of the golang receiver struct to be instrumented, ie for the "net/http" package, "response" is a receiver struct
	// ReceiverName is disallowed if FunctionName is provided; Receiver must be a concrete type (not an interface)
	ReceiverName string `json:"receiverName,omitempty" yaml:"receiverName,omitempty"`
	// ReceiverMethodName is the name of the golang method, given a receiver struct, to be instrumented
	// for example for "net/http" package, "response" is a receiver struct and "WriteHeader" is a method of that struct
	// ReceiverMethodName is mandatory if ReceiverName is provided, and disallowed if FunctionName is provided
	ReceiverMethodName string `json:"receiverMethodName,omitempty" yaml:"receiverMethodName,omitempty"`
}

// GolangCustomProbeWithMetadata is a Go probe plus rule-only metadata.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type GolangCustomProbeWithMetadata struct {
	GolangCustomProbe   `json:",inline"`
	CustomProbeMetadata `json:",inline"`
}

// For golang we require package name and either function name or receiver name + method name
func (gcp *GolangCustomProbe) Verify() error {
	switch {
	case gcp.PackageName == "":
		return errors.New("package name is required")
	case gcp.FunctionName != "" && gcp.ReceiverName != "" && gcp.ReceiverMethodName != "":
		return errors.New("too many arguments; either function name or receiver name + method name are required")
	case gcp.FunctionName == "" && gcp.ReceiverName == "" && gcp.ReceiverMethodName == "":
		return errors.New("too few arguments; either function name or receiver name + method name are required")
	case (gcp.ReceiverName == "" && gcp.ReceiverMethodName != "") || (gcp.ReceiverName != "" && gcp.ReceiverMethodName == ""):
		return errors.New("both receiver name and receiver method name are required when using receiver methods")
	default:
		return nil
	}
}

func (gcp *GolangCustomProbeWithMetadata) Verify() error {
	if err := gcp.CustomProbeMetadata.Verify(); err != nil {
		return err
	}
	return gcp.GolangCustomProbe.Verify()
}

type CppCustomProbe struct {
	// function signature to add custom probe for.
	// the following scenarios are possible:
	// namespace::class::function - for example std::vector::push_back - all overloads of this function
	// namespace::function - targeting a function (and its possible overloads) in a specific namespac
	// function - for C-like functions a single function name can be used: e.g SSL_write
	Signature string `json:"signature"`
}

// CppCustomProbeWithMetadata is a C++ probe plus rule-only metadata.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CppCustomProbeWithMetadata struct {
	CppCustomProbe      `json:",inline"`
	CustomProbeMetadata `json:",inline"`
}

func (c *CppCustomProbe) Verify() error {
	if c.Signature == "" {
		return errors.New("signature is required")
	}

	return nil
}

func (c *CppCustomProbeWithMetadata) Verify() error {
	if err := c.CustomProbeMetadata.Verify(); err != nil {
		return err
	}
	return c.CppCustomProbe.Verify()
}

// PhpCustomProbe contains the details for a custom probe for PHP applications.
// className is optional; when empty, functionName refers to a global function.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PhpCustomProbe struct {
	ClassName    string `json:"className,omitempty" yaml:"className,omitempty"`
	FunctionName string `json:"functionName" yaml:"functionName"`
}

// PhpCustomProbeWithMetadata is a PHP probe plus rule-only metadata.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PhpCustomProbeWithMetadata struct {
	PhpCustomProbe      `json:",inline"`
	CustomProbeMetadata `json:",inline"`
}

func (pcp *PhpCustomProbe) Verify() error {
	if pcp.FunctionName == "" {
		return errors.New("function name is required")
	}
	return nil
}

func (pcp *PhpCustomProbeWithMetadata) Verify() error {
	if err := pcp.CustomProbeMetadata.Verify(); err != nil {
		return err
	}
	return pcp.PhpCustomProbe.Verify()
}

func (pcp *PhpCustomProbe) String() string {
	if pcp.ClassName == "" {
		return pcp.FunctionName
	}
	return fmt.Sprintf("%s::%s", pcp.ClassName, pcp.FunctionName)
}
