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
	// the library name shouold be the same as the "instrumentation scope name" that can be found on the generate span attributes.
	DisabledInstrumentationLibraries []InstrumentationLibrary `json:"disabledInstrumentationLibraries,omitempty"`

	// for instrumentation libraries that are disabled by default, this field can be used to enable them.
	// the list of such libraries is small and depends on the language and agent type.
	// common example: nodejs fs, dns, net instrumentations are disabled by default and can be opt-in for trace collection.
	EnabledInstrumentationLibraries []InstrumentationLibrary `json:"enabledInstrumentationLibraries,omitempty"`
}
