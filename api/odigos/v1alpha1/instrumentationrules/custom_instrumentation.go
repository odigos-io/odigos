package instrumentationrules

import (
	"errors"
	"fmt"
)

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type CustomInstrumentations struct {
	Golang []GolangCustomProbe `json:"golang,omitempty"`
	Java   []JavaCustomProbe   `json:"java,omitempty"`
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
			return fmt.Errorf("invalid custom instrumentation golang: %w", err)
		}
	}
	// Validate Java probes
	for _, p := range ci.Java {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid custom instrumentation java: %w", err)
		}
	}
	return nil
}

// java custom probe contains the details for a custom probe for java applications,
// which includes the class name and method name to be instrumented.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type JavaCustomProbe struct {
	ClassName  string `json:"className,omitempty"`
	MethodName string `json:"methodName,omitempty"`
}

// For java we always require both class name and method name
func (jcp *JavaCustomProbe) Verify() error {
	if jcp.ClassName == "" {
		return fmt.Errorf("className is required for JavaCustomProbe")
	}
	if jcp.MethodName == "" {
		return fmt.Errorf("methodName is required for JavaCustomProbe")
	}
	return nil
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
	// Package name is required
	PackageName        string `json:"packageName"`
	FunctionName       string `json:"functionName,omitempty"`
	ReceiverName       string `json:"receiverName,omitempty"`
	ReceiverMethodName string `json:"receiverMethodName,omitempty"`
}

// For golang we require package name and either function name or receiver name + method name
func (gcp *GolangCustomProbe) Verify() error {
	if gcp.PackageName == "" {
		return errors.New("packageName is required for GolangCustomProbe")
	}
	if gcp.FunctionName == "" || (gcp.ReceiverName == "" && gcp.ReceiverMethodName == "") {
		return errors.New("either functionName or receiver name + method name is required for GolangCustomProbe")
	}
	return nil
}
