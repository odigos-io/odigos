package describe

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/odigos-io/odigos/k8sutils/pkg/describe/properties"
)

const (
	redPrefix    = "\033[31m"
	greenPrefix  = "\033[32m"
	yellowPrefix = "\033[33m"
	colorSuffix  = "\033[0m"
)

func TestDescribeText(t *testing.T) {
	testCases := []struct {
		name      string
		indent    int
		isListKey bool
		format    string
		args      []interface{}
		expected  string
	}{
		{
			name:     "no indentation",
			format:   "Odigos Pipeline:",
			expected: "Odigos Pipeline:\n",
		},
		{
			name:     "two spaces per indentation level",
			indent:   3,
			format:   "Ready: true",
			expected: "      Ready: true\n",
		},
		{
			name:     "format arguments are applied",
			indent:   1,
			format:   "Pods (Total %d, %s):",
			args:     []interface{}{2, "Running 2"},
			expected: "  Pods (Total 2, Running 2):\n",
		},
		{
			name:      "a list key is prefixed with a dash one level up",
			indent:    2,
			isListKey: true,
			format:    "Container Name: checkout",
			expected:  "  - Container Name: checkout\n",
		},
		{
			// the list key indent must not become negative
			name:      "a list key at the top level",
			indent:    0,
			isListKey: true,
			format:    "Pod Name: checkout-1",
			expected:  "- Pod Name: checkout-1\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder

			describeText(&sb, tc.indent, tc.isListKey, tc.format, tc.args...)

			assert.Equal(t, tc.expected, sb.String())
		})
	}
}

func TestDescribeTextAppendsToTheBuilder(t *testing.T) {
	var sb strings.Builder

	describeText(&sb, 0, false, "first")
	describeText(&sb, 1, false, "second")

	assert.Equal(t, "first\n  second\n", sb.String())
}

func TestPrintProperty(t *testing.T) {
	testCases := []struct {
		name     string
		property *properties.EntityProperty
		expected string
	}{
		{
			name:     "a nil property prints nothing",
			property: nil,
			expected: "",
		},
		{
			name:     "a property without a status is not colored",
			property: &properties.EntityProperty{Name: "Odigos Version", Value: "v1.22.0"},
			expected: "  Odigos Version: v1.22.0\n",
		},
		{
			name: "a successful property is green",
			property: &properties.EntityProperty{
				Name: "Ready", Value: true, Status: properties.PropertyStatusSuccess,
			},
			expected: "  " + greenPrefix + "Ready: true" + colorSuffix + "\n",
		},
		{
			name: "an erroneous property is red",
			property: &properties.EntityProperty{
				Name: "Failed Replicas", Value: 2, Status: properties.PropertyStatusError,
			},
			expected: "  " + redPrefix + "Failed Replicas: 2" + colorSuffix + "\n",
		},
		{
			name: "a transitioning property is yellow",
			property: &properties.EntityProperty{
				Name: "Deployed", Value: false, Status: properties.PropertyStatusTransitioning,
			},
			expected: "  " + yellowPrefix + "Deployed: false" + colorSuffix + "\n",
		},
		{
			name: "a list key property is prefixed with a dash",
			property: &properties.EntityProperty{
				Name: "Container Name", Value: "checkout", ListKey: true,
			},
			expected: "- Container Name: checkout\n",
		},
		{
			name: "a colored list key property keeps both the dash and the color",
			property: &properties.EntityProperty{
				Name: "Healthy", Value: true, Status: properties.PropertyStatusSuccess, ListKey: true,
			},
			expected: "- " + greenPrefix + "Healthy: true" + colorSuffix + "\n",
		},
		{
			name: "a slice value is rendered by its default formatting",
			property: &properties.EntityProperty{
				Name: "Actual Devices", Value: []string{"golang-community"},
			},
			expected: "  Actual Devices: [golang-community]\n",
		},
		{
			// the explanation is metadata for the graphql api and is not part of the text output
			name: "the explanation is not printed",
			property: &properties.EntityProperty{
				Name: "Ready", Value: true, Explain: "ready means the collector can accept data",
			},
			expected: "  Ready: true\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var sb strings.Builder

			printProperty(&sb, 1, tc.property)

			assert.Equal(t, tc.expected, sb.String())
		})
	}
}

func TestWrapTextInColor(t *testing.T) {
	assert.Equal(t, redPrefix+"error"+colorSuffix, wrapTextInRed("error"))
	assert.Equal(t, greenPrefix+"success"+colorSuffix, wrapTextInGreen("success"))
	assert.Equal(t, yellowPrefix+"transitioning"+colorSuffix, wrapTextInYellow("transitioning"))
}
