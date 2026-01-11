package distros

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestCommunityDefaulter_GetDefaultDistroNames(t *testing.T) {
	defaulter := NewCommunityDefaulter()
	distroNames := defaulter.GetDefaultDistroNames()

	expectedLanguages := []common.ProgrammingLanguage{
		common.JavascriptProgrammingLanguage,
		common.PythonProgrammingLanguage,
		common.DotNetProgrammingLanguage,
		common.JavaProgrammingLanguage,
		common.GoProgrammingLanguage,
		common.PhpProgrammingLanguage,
		common.RubyProgrammingLanguage,
		common.RustProgrammingLanguage,
	}

	for _, lang := range expectedLanguages {
		if _, ok := distroNames[lang]; !ok {
			t.Errorf("Expected distro for language %s not found", lang)
		}
	}
}

func TestNewCommunityGetter(t *testing.T) {
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("Failed to create community getter: %v", err)
	}

	distros := getter.GetAllDistros()
	if len(distros) == 0 {
		t.Error("Expected at least one distro, got none")
	}

	distroNames := make(map[string]bool)
	for _, d := range distros {
		distroNames[d.Name] = true
	}

	expectedDistros := []string{
		"rust-community",
		"rust-native",
		"golang-community",
		"python-community",
		"nodejs-community",
		"java-community",
		"dotnet-community",
		"php-community",
		"ruby-community",
	}

	for _, name := range expectedDistros {
		if !distroNames[name] {
			t.Errorf("Expected distro %s not found", name)
		}
	}
}

func TestRustNativeDistro(t *testing.T) {
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("Failed to create community getter: %v", err)
	}

	rustNative := getter.GetDistroByName("rust-native")
	if rustNative == nil {
		t.Fatal("rust-native distro not found")
	}

	if rustNative.Language != common.RustProgrammingLanguage {
		t.Errorf("Expected language rust, got %s", rustNative.Language)
	}

	if rustNative.DisplayName == "" {
		t.Error("DisplayName should not be empty")
	}

	if rustNative.Description == "" {
		t.Error("Description should not be empty")
	}

	if len(rustNative.RuntimeEnvironments) == 0 {
		t.Error("Expected at least one runtime environment")
	}

	if !rustNative.EnvironmentVariables.OtlpHttpLocalNode {
		t.Error("Expected OtlpHttpLocalNode to be true for rust-native")
	}

	if !rustNative.EnvironmentVariables.SignalsAsStaticOtelEnvVars {
		t.Error("Expected SignalsAsStaticOtelEnvVars to be true for rust-native")
	}

	staticVars := rustNative.EnvironmentVariables.StaticVariables
	if len(staticVars) == 0 {
		t.Error("Expected static environment variables for rust-native")
	}

	expectedEnvVars := map[string]string{
		"OTEL_EXPORTER_OTLP_PROTOCOL": "grpc",
		"OTEL_TRACES_EXPORTER":        "otlp",
		"OTEL_METRICS_EXPORTER":       "otlp",
		"OTEL_LOGS_EXPORTER":          "otlp",
		"OTEL_PROPAGATORS":            "tracecontext,baggage",
	}

	foundVars := make(map[string]string)
	for _, v := range staticVars {
		foundVars[v.EnvName] = v.EnvValue
	}

	for name, expectedValue := range expectedEnvVars {
		if value, ok := foundVars[name]; !ok {
			t.Errorf("Expected environment variable %s not found", name)
		} else if value != expectedValue {
			t.Errorf("Environment variable %s: expected %s, got %s", name, expectedValue, value)
		}
	}

	if rustNative.RuntimeAgent == nil {
		t.Error("Expected RuntimeAgent to be set for rust-native")
	} else {
		if !rustNative.RuntimeAgent.K8sAttrsViaEnvVars {
			t.Error("Expected K8sAttrsViaEnvVars to be true")
		}
	}
}

func TestRustCommunityDistro(t *testing.T) {
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("Failed to create community getter: %v", err)
	}

	rustCommunity := getter.GetDistroByName("rust-community")
	if rustCommunity == nil {
		t.Fatal("rust-community distro not found")
	}

	if rustCommunity.Language != common.RustProgrammingLanguage {
		t.Errorf("Expected language rust, got %s", rustCommunity.Language)
	}

	if rustCommunity.RuntimeAgent == nil {
		t.Error("Expected RuntimeAgent to be set for rust-community")
	} else {
		if !rustCommunity.RuntimeAgent.NoRestartRequired {
			t.Error("Expected NoRestartRequired to be true for eBPF-based rust-community")
		}
	}
}

func TestNewProvider(t *testing.T) {
	defaulter := NewCommunityDefaulter()
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("Failed to create community getter: %v", err)
	}

	provider, err := NewProvider(defaulter, getter)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	distroNames := provider.GetDefaultDistroNames()
	rustDistro := distroNames[common.RustProgrammingLanguage]
	if rustDistro == "" {
		t.Error("No default distro for Rust language")
	}

	distro := provider.GetDistroByName(rustDistro)
	if distro == nil {
		t.Errorf("Default Rust distro %s not found in provider", rustDistro)
	}
}

func TestAllDistrosHaveRequiredFields(t *testing.T) {
	getter, err := NewCommunityGetter()
	if err != nil {
		t.Fatalf("Failed to create community getter: %v", err)
	}

	for _, distro := range getter.GetAllDistros() {
		t.Run(distro.Name, func(t *testing.T) {
			if distro.Name == "" {
				t.Error("Distro name is empty")
			}
			if distro.DisplayName == "" {
				t.Error("Distro display name is empty")
			}
			if distro.Language == "" {
				t.Error("Distro language is empty")
			}
			if distro.Description == "" {
				t.Error("Distro description is empty")
			}
			if len(distro.RuntimeEnvironments) == 0 {
				t.Error("Distro has no runtime environments")
			}
		})
	}
}

