package urltemplate

import "testing"

func TestSegmentTemplateName(t *testing.T) {
	tests := []struct {
		segment string
		want    string
	}{
		{"users", ""},
		{"1234567", "id"},
		{"123e4567-e89b-12d3-a456-426614174000", "id"},
		{"0123456789abcdef", "id"},
		{"2025-12-04", "date"},
		{"user@example.com", "email"},
		{"INC0012686", "id"},
	}
	for _, tt := range tests {
		if got := SegmentTemplateName(tt.segment); got != tt.want {
			t.Errorf("SegmentTemplateName(%q) = %q, want %q", tt.segment, got, tt.want)
		}
	}
}
