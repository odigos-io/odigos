package containers

import "testing"

func TestGetName(t *testing.T) {
	tests := []struct {
		testName string
		prefix   string
		name     string
		version  string
		expected string
	}{
		{
			testName: "no prefix",
			prefix:   "",
			name:     "test-component",
			version:  "v1.0.0",
			expected: "test-component:v1.0.0",
		},
		{
			testName: "with prefix",
			prefix:   "docker.io/keyval",
			name:     "test-component",
			version:  "v1.0.0",
			expected: "docker.io/keyval/test-component:v1.0.0",
		},
		{
			testName: "prefix with trailing slash",
			prefix:   "docker.io/keyval/",
			name:     "test-component",
			version:  "v1.0.0",
			expected: "docker.io/keyval/test-component:v1.0.0",
		},
		{
			testName: "pinned image SHA",
			prefix:   "",
			name:     "test-component@SHA256:12345",
			version:  "",
			expected: "test-component@SHA256:12345",
		},
		{
			testName: "pinned image SHA ignores passed version",
			prefix:   "",
			name:     "test-component@SHA256:12345",
			version:  "v1.0.0",
			expected: "test-component@SHA256:12345",
		},
		{
			testName: "pinned image SHA with prefix",
			prefix:   "docker.io/keyval",
			name:     "test-component@SHA256:12345",
			version:  "v1.0.0",
			expected: "docker.io/keyval/test-component@SHA256:12345",
		},
		{
			testName: "no prefix, but prefix is in image name",
			prefix:   "",
			name:     "docker.io/keyval/test-component",
			version:  "v1.0.0",
			expected: "docker.io/keyval/test-component:v1.0.0",
		},
		{
			testName: "image name with tag isn't overwritten",
			prefix:   "",
			name:     "docker.io/keyval/test-component:v1.0.0",
			version:  "v1.0.1",
			expected: "docker.io/keyval/test-component:v1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			got := GetImageName(tt.prefix, tt.name, tt.version)
			if got != tt.expected {
				t.Errorf("Test '%s' failed: input=%+v, expected=%s, actual=%s", tt.name, tt, tt.expected, got)
			}
		})
	}
}
