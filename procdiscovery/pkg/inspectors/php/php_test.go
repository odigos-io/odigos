package php

import (
	"testing"
)

func Test_normalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"8", "8"},
		{"82", "8.2"},
		{"834", "8.3.4"},
		{"8.3", "8.3"},
		{"8.3.4", "8.3.4"},
		{"7.4.33", "7.4.33"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q; want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_joinNonEmpty(t *testing.T) {
	tests := []struct {
		parts    []string
		expected string
	}{
		{[]string{"8", "3", "4"}, "8.3.4"},
		{[]string{"8", "3", ""}, "8.3"},
		{[]string{"8", "", ""}, "8"},
		{[]string{"", "", ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := joinNonEmpty(tt.parts...)
			if result != tt.expected {
				t.Errorf("joinNonEmpty(%v) = %q; want %q", tt.parts, result, tt.expected)
			}
		})
	}
}

func Test_phpExecutableRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"php", true},
		{"php-cgi", true},
		{"php-fpm", true},
		{"php82", true},
		{"php8.3", true},
		{"php-fpm82", true},
		{"php-fpm8.3", true},
		{"php-cgi74", true},
		{"php-cgi7.4", true},
		{"php7.4.33", true},
		{"python", false},
		{"phpstorm", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := phpExecutableRegex.MatchString(tt.input)
			if result != tt.expected {
				t.Errorf("phpExecutableRegex.MatchString(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func Test_phpSoVersionRegex(t *testing.T) {
	tests := []struct {
		input          string
		shouldMatch    bool
		expectedGroups []string
	}{
		{"libphp.so.8.3", true, []string{"php.so.8.3", "", "8", "3", ""}},
		{"libphp.so.8.3.4", true, []string{"php.so.8.3.4", "", "8", "3", "4"}},
		{"/usr/lib/apache2/mod_php82.so", true, []string{"php82.so", "82", "", "", ""}},
		{"/usr/lib/apache2/mod_php8.so", true, []string{"php8.so", "8", "", "", ""}},
		{"libphp8.so", true, []string{"php8.so", "8", "", "", ""}},
		{"/usr/lib/php8.2.so", true, []string{"php8.2.so", "8.2", "", "", ""}},
		{"libphp8.2.so", true, []string{"php8.2.so", "8.2", "", "", ""}},
		{"libphp.so", true, []string{"php.so", "", "", "", ""}},
		{"libpython.so", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := phpSoVersionRe.FindStringSubmatch(tt.input)
			if tt.shouldMatch {
				if matches == nil {
					t.Errorf("expected to match %q but didn't", tt.input)
					return
				}
				for i, expected := range tt.expectedGroups {
					if i < len(matches) && matches[i] != expected {
						t.Errorf("group %d: got %q, want %q", i, matches[i], expected)
					}
				}
			} else if matches != nil {
				t.Errorf("should not match %q but got %v", tt.input, matches)
			}
		})
	}
}

func Test_phpPathVersionRegex(t *testing.T) {
	tests := []struct {
		input          string
		shouldMatch    bool
		expectedGroups []string
	}{
		{"/usr/lib/php/8.3/", true, []string{"/php/8.3/", "8.3"}},
		{"/usr/lib/php/8.3.4/", true, []string{"/php/8.3.4/", "8.3.4"}},
		{"/usr/lib/php/8.3.4/something", true, []string{"/php/8.3.4/", "8.3.4"}},
		{"/var/lib/php8.2/something", true, []string{"/php8.2/", "8.2"}},
		{"/var/lib/php82/modules", true, []string{"/php82/", "82"}},
		{"/php/8/", true, []string{"/php/8/", "8"}},
		{"/ruby/3.2.0/", false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			matches := phpPathVersionRe.FindStringSubmatch(tt.input)
			if tt.shouldMatch {
				if matches == nil {
					t.Errorf("expected to match %q but didn't", tt.input)
					return
				}
				for i, expected := range tt.expectedGroups {
					if i < len(matches) && matches[i] != expected {
						t.Errorf("group %d: got %q, want %q", i, matches[i], expected)
					}
				}
			} else if matches != nil {
				t.Errorf("should not match %q but got %v", tt.input, matches)
			}
		})
	}
}

func Test_isBetterVersion(t *testing.T) {
	tests := []struct {
		newVer      string
		currentBest string
		expected    bool
	}{
		{"8.3", "", true},
		{"8.3.4", "8.3", true},
		{"8.3", "7.4", false},
		{"8", "8.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.newVer+"_vs_"+tt.currentBest, func(t *testing.T) {
			result := isBetterVersion(tt.newVer, tt.currentBest)
			if result != tt.expected {
				t.Errorf("isBetterVersion(%q, %q) = %v; want %v", tt.newVer, tt.currentBest, result, tt.expected)
			}
		})
	}
}
