package urltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantSegments     []string
		wantLeadingSlash bool
	}{
		{
			name:             "absolute path",
			path:             "/users/123",
			wantSegments:     []string{"users", "123"},
			wantLeadingSlash: true,
		},
		{
			name:             "relative path",
			path:             "users/123",
			wantSegments:     []string{"users", "123"},
			wantLeadingSlash: false,
		},
		{
			name:             "single segment",
			path:             "/health",
			wantSegments:     []string{"health"},
			wantLeadingSlash: true,
		},
		{
			name:             "root path",
			path:             "/",
			wantSegments:     []string{""},
			wantLeadingSlash: true,
		},
		{
			name:             "empty path",
			path:             "",
			wantSegments:     []string{""},
			wantLeadingSlash: false,
		},
		{
			name:             "trailing slash keeps an empty last segment",
			path:             "/users/",
			wantSegments:     []string{"users", ""},
			wantLeadingSlash: true,
		},
		{
			name:             "only the first slash is consumed",
			path:             "//users",
			wantSegments:     []string{"", "users"},
			wantLeadingSlash: true,
		},
		{
			name:             "empty inner segment is preserved",
			path:             "/users//123",
			wantSegments:     []string{"users", "", "123"},
			wantLeadingSlash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, hadLeadingSlash := SplitPath(tt.path)
			assert.Equal(t, tt.wantSegments, segments)
			assert.Equal(t, tt.wantLeadingSlash, hadLeadingSlash)
		})
	}
}
