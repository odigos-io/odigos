package instrumentationrules

import "github.com/odigos-io/odigos/common"

type InstrumentationLibrary struct {

	// the programming language of the relevant library
	ProgrammingLanguage common.ProgrammingLanguage `json:"programmingLanguage"`

	// the name of the library to configure. required.
	// exact syntax and format depends on the programming language.
	LibraryName string `json:"libraryName"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true

type TraceVerbosity struct {

	// instrumentation libraries to configure
	DisabledInstrumentationLibraries []InstrumentationLibrary `json:"disabledInstrumentationLibraries,omitempty"`
}
