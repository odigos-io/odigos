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
			return fmt.Errorf("invalid configuration for golang custom instrumentation: %w", err)
		}
	}
	// Validate Java probes
	for _, p := range ci.Java {
		if err := p.Verify(); err != nil {
			return fmt.Errorf("invalid configuration for java custom instrumentation: %w", err)
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
		return errors.New("class name is required")
	}
	if jcp.MethodName == "" {
		return errors.New("method Name is required")
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
	// PackageName is the name of the golang pacakge (ie net/http); Package name is always required
	PackageName string `json:"packageName"`
	// FunctionName is the name of the golang function to be instrumented, ie net/http.ListenAndServe
	// Function name is optional if ReceiverName and ReceiverMethodName are provided
	FunctionName string `json:"functionName,omitempty"`
	// ReceiverName is the name of the golang receiver struct to be instrumented, ie http.Server
	// ReceiverName is optional if FunctionName is provided
	ReceiverName string `json:"receiverName,omitempty"`
	// ReceiverMethodName is the name of the golang method, given a receiver struct, to be instrumented
	// for example for http.Server.ListenAndServe, ReceiverMethodName is ListenAndServe
	// ReceiverMethodName is optional if FunctionName is provided but required if ReceiverName is provided
	ReceiverMethodName string `json:"receiverMethodName,omitempty"`
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
