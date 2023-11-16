package common

type OdigosInstrumentationDevice string

const (
	JavaDeviceName       OdigosInstrumentationDevice = "instrumentation.odigos.io/java"
	PythonDeviceName     OdigosInstrumentationDevice = "instrumentation.odigos.io/python"
	DotNetDeviceName     OdigosInstrumentationDevice = "instrumentation.odigos.io/dotnet"
	JavascriptDeviceName OdigosInstrumentationDevice = "instrumentation.odigos.io/javascript"
)

var InstrumentationDevices = []OdigosInstrumentationDevice{
	JavaDeviceName,
	PythonDeviceName,
	DotNetDeviceName,
	JavascriptDeviceName,
}

func ProgrammingLanguageToInstrumentationDevice(language ProgrammingLanguage) OdigosInstrumentationDevice {
	switch language {
	case JavaProgrammingLanguage:
		return JavaDeviceName
	case PythonProgrammingLanguage:
		return PythonDeviceName
	case DotNetProgrammingLanguage:
		return DotNetDeviceName
	case JavascriptProgrammingLanguage:
		return JavascriptDeviceName
	default:
		return ""
	}
}
