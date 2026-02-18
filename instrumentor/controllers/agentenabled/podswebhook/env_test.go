package podswebhook

import (
	"testing"
	"text/template"

	"github.com/odigos-io/odigos/distros/distro"
	corev1 "k8s.io/api/core/v1"
)

func parseTemplate(t *testing.T, text string) *template.Template {
	t.Helper()
	tmpl, err := template.New("test").Option("missingkey=error").Parse(text)
	if err != nil {
		t.Fatalf("failed to parse template %q: %v", text, err)
	}
	return tmpl
}

func findEnvVar(container *corev1.Container, name string) *corev1.EnvVar {
	for i := range container.Env {
		if container.Env[i].Name == name {
			return &container.Env[i]
		}
	}
	return nil
}

func TestAppendEnvVar_NoExisting_NoCRI(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	result, err := appendEnvVarToPodContainer(existing, container, ev, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result["PHP_INI_SCAN_DIR"]; !ok {
		t.Error("expected PHP_INI_SCAN_DIR in returned env names map")
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got == nil {
		t.Fatal("expected PHP_INI_SCAN_DIR to be set on container")
	}
	if got.Value != ":/var/odigos/php/8.2" {
		t.Errorf("expected ':/var/odigos/php/8.2', got %q", got.Value)
	}
}

func TestAppendEnvVar_NoExisting_WithCRI(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}
	params := map[string]string{
		"PHP_INI_SCAN_DIR": "/etc/php/custom-ini",
	}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got == nil {
		t.Fatal("expected PHP_INI_SCAN_DIR to be set on container")
	}
	if got.Value != "/etc/php/custom-ini:/var/odigos/php/8.2" {
		t.Errorf("expected '/etc/php/custom-ini:/var/odigos/php/8.2', got %q", got.Value)
	}
}

func TestAppendEnvVar_ExistingInManifest_Appends(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "PHP_INI_SCAN_DIR", Value: "/my/custom/dir"},
		},
	}
	existing := EnvVarNamesMap{"PHP_INI_SCAN_DIR": {}}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != "/my/custom/dir:/var/odigos/php/8.2" {
		t.Errorf("expected '/my/custom/dir:/var/odigos/php/8.2', got %q", got.Value)
	}
}

func TestAppendEnvVar_ExistingWithLeadingColon_Appends(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "PHP_INI_SCAN_DIR", Value: ":/my/custom/dir"},
		},
	}
	existing := EnvVarNamesMap{"PHP_INI_SCAN_DIR": {}}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != ":/my/custom/dir:/var/odigos/php/8.2" {
		t.Errorf("expected ':/my/custom/dir:/var/odigos/php/8.2', got %q", got.Value)
	}
}

func TestAppendEnvVar_Idempotent_ManifestAlreadyHasValue(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "PHP_INI_SCAN_DIR", Value: "/my/dir:/var/odigos/php/8.2"},
		},
	}
	existing := EnvVarNamesMap{"PHP_INI_SCAN_DIR": {}}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != "/my/dir:/var/odigos/php/8.2" {
		t.Errorf("value should be unchanged, got %q", got.Value)
	}
}

func TestAppendEnvVar_Idempotent_CRIAlreadyHasValue(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}
	params := map[string]string{
		"PHP_INI_SCAN_DIR": "/image/path:/var/odigos/php/8.2",
	}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != ":/var/odigos/php/8.2" {
		t.Errorf("CRI already contains our value, should not double-prepend, got %q", got.Value)
	}
}

func TestAppendEnvVar_WithTemplate(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "PHP_INI_SCAN_DIR", Value: "/user/ini"},
		},
	}
	existing := EnvVarNamesMap{"PHP_INI_SCAN_DIR": {}}
	params := map[string]string{
		"RUNTIME_VERSION_MAJOR_MINOR": "8.3",
	}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/{{.RUNTIME_VERSION_MAJOR_MINOR}}",
		AppendToExisting: true,
		Template:         parseTemplate(t, ":/var/odigos/php/{{.RUNTIME_VERSION_MAJOR_MINOR}}"),
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != "/user/ini:/var/odigos/php/8.3" {
		t.Errorf("expected '/user/ini:/var/odigos/php/8.3', got %q", got.Value)
	}
}

func TestAppendEnvVar_TemplateWithCRI_NoManifest(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}
	params := map[string]string{
		"RUNTIME_VERSION_MAJOR_MINOR": "8.2",
		"PHP_INI_SCAN_DIR":           "/image/ini",
	}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/{{.RUNTIME_VERSION_MAJOR_MINOR}}",
		AppendToExisting: true,
		Template:         parseTemplate(t, ":/var/odigos/php/{{.RUNTIME_VERSION_MAJOR_MINOR}}"),
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != "/image/ini:/var/odigos/php/8.2" {
		t.Errorf("expected '/image/ini:/var/odigos/php/8.2', got %q", got.Value)
	}
}

func TestAppendEnvVar_TemplateError(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}
	tmpl := parseTemplate(t, "{{.MISSING_KEY}}")

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		AppendToExisting: true,
		Template:         tmpl,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, map[string]string{})
	if err == nil {
		t.Error("expected template execution error for missing key")
	}
}

func TestAppendEnvVar_EmptyCRIValue_Ignored(t *testing.T) {
	container := &corev1.Container{Name: "app"}
	existing := EnvVarNamesMap{}
	params := map[string]string{
		"PHP_INI_SCAN_DIR": "",
	}

	ev := distro.StaticEnvironmentVariable{
		EnvName:          "PHP_INI_SCAN_DIR",
		EnvValue:         ":/var/odigos/php/8.2",
		AppendToExisting: true,
	}

	_, err := appendEnvVarToPodContainer(existing, container, ev, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if got.Value != ":/var/odigos/php/8.2" {
		t.Errorf("empty CRI value should be ignored, got %q", got.Value)
	}
}

func TestInjectStaticEnvVars_MixedAppendAndNormal(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "PHP_INI_SCAN_DIR", Value: "/existing"},
		},
	}
	existing := GetEnvVarNamesSet(container)
	params := map[string]string{}

	envVars := []distro.StaticEnvironmentVariable{
		{
			EnvName:          "PHP_INI_SCAN_DIR",
			EnvValue:         ":/var/odigos/php/8.2",
			AppendToExisting: true,
		},
		{
			EnvName:  "OTEL_PHP_AUTOLOAD_ENABLED",
			EnvValue: "true",
		},
		{
			EnvName:  "ALREADY_SET",
			EnvValue: "should-be-skipped",
		},
	}

	container.Env = append(container.Env, corev1.EnvVar{Name: "ALREADY_SET", Value: "original"})
	existing["ALREADY_SET"] = struct{}{}

	result, err := InjectStaticEnvVarsToPodContainer(existing, container, envVars, params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	scanDir := findEnvVar(container, "PHP_INI_SCAN_DIR")
	if scanDir.Value != "/existing:/var/odigos/php/8.2" {
		t.Errorf("PHP_INI_SCAN_DIR: expected '/existing:/var/odigos/php/8.2', got %q", scanDir.Value)
	}

	autoload := findEnvVar(container, "OTEL_PHP_AUTOLOAD_ENABLED")
	if autoload == nil || autoload.Value != "true" {
		t.Errorf("OTEL_PHP_AUTOLOAD_ENABLED: expected 'true', got %v", autoload)
	}
	if _, ok := result["OTEL_PHP_AUTOLOAD_ENABLED"]; !ok {
		t.Error("OTEL_PHP_AUTOLOAD_ENABLED should be in returned env names map")
	}

	alreadySet := findEnvVar(container, "ALREADY_SET")
	if alreadySet.Value != "original" {
		t.Errorf("ALREADY_SET should be unchanged, got %q", alreadySet.Value)
	}
}

func TestInjectStaticEnvVars_AppendFalse_SkipsExisting(t *testing.T) {
	container := &corev1.Container{
		Name: "app",
		Env: []corev1.EnvVar{
			{Name: "MY_VAR", Value: "original"},
		},
	}
	existing := GetEnvVarNamesSet(container)

	envVars := []distro.StaticEnvironmentVariable{
		{
			EnvName:          "MY_VAR",
			EnvValue:         "new-value",
			AppendToExisting: false,
		},
	}

	_, err := InjectStaticEnvVarsToPodContainer(existing, container, envVars, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := findEnvVar(container, "MY_VAR")
	if got.Value != "original" {
		t.Errorf("non-append var should keep original value, got %q", got.Value)
	}
}
