package utils

import (
	"testing"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func TestIsProcessEqualProcessNamesWithVersion(t *testing.T) {
	names := []string{"ruby", "puma", "python"}

	cases := []struct {
		exePath string
		want    bool
	}{
		{"/usr/bin/ruby", true},            // exact
		{"/usr/bin/ruby3.3", true},         // versioned (CORE-1033)
		{"/usr/local/bin/ruby3.4.4", true}, // versioned patch
		{"/usr/bin/python3.11", true},      // versioned python
		{"/usr/bin/puma", true},            // exact
		{"/usr/bin/rubyfoo", false},        // non-version suffix must not match
		{"/usr/bin/node", false},           // unrelated
		{"/usr/bin/ruby-lsp", false},       // suffix is not a version
	}

	for _, c := range cases {
		pcx := &process.ProcessContext{Details: process.Details{ExePath: c.exePath}}
		if got := IsProcessEqualProcessNamesWithVersion(pcx, names); got != c.want {
			t.Errorf("IsProcessEqualProcessNamesWithVersion(%q) = %v, want %v", c.exePath, got, c.want)
		}
	}
}
