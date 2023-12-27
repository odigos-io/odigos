package common

type ConfigOption struct {
	OptionKey string `json:"optionKey"`
	SpanKind  string `json:"spanKind,omitempty"`
}

type InstrumentationLibrary struct {
	LibraryName string `json:"libraryName"`
	Options []ConfigOption `json:"options"`
}


type OptionByContainer struct {
	ContainerName string `json:"containerName"`
	InstrumentationLibraries []InstrumentationLibrary `json:"instrumentationLibraryName"`
}
