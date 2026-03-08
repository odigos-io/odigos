package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv_1_4_0 "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func TestMatchServerAddress(t *testing.T) {
	tests := []struct {
		name              string
		attrs             map[string]string
		ruleServerAddress string
		wantMatch         bool
	}{
		{
			name: "match",
			attrs: map[string]string{
				string(semconv.ServerAddressKey): "api.example.com",
			},
			ruleServerAddress: "api.example.com",
			wantMatch:         true,
		},
		{
			name: "no match when address differs",
			attrs: map[string]string{
				string(semconv.ServerAddressKey): "api.example.com",
			},
			ruleServerAddress: "other.example.com",
			wantMatch:         false,
		},
		{
			name:              "no match when server.address missing",
			attrs:             map[string]string{"other.attr": "value"},
			ruleServerAddress: "api.example.com",
			wantMatch:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			got := matchServerAddress(span, tt.ruleServerAddress)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func TestMatchHttpRoute(t *testing.T) {
	tests := []struct {
		name            string
		attrs           map[string]string
		ruleRouteExact  string
		ruleRoutePrefix string
		wantMatch       bool
	}{
		{
			name:            "both rules unset match any",
			attrs:           map[string]string{},
			ruleRouteExact:  "",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
		{
			name: "match via http.route",
			attrs: map[string]string{
				string(semconv.HTTPRouteKey): "/users/:id",
			},
			ruleRouteExact:  "/users/:id",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
		{
			name: "no match via http.route",
			attrs: map[string]string{
				string(semconv.HTTPRouteKey): "/users/:id",
			},
			ruleRouteExact:  "/orders",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
		{
			name: "match via url.path when http.route absent",
			attrs: map[string]string{
				string(semconv.URLPathKey): "/api/v1/health",
			},
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       true,
		},
		{
			name: "match via http.target when http.route and url.path absent",
			attrs: map[string]string{
				string(semconv_1_4_0.HTTPTargetKey): "/legacy/path",
			},
			ruleRouteExact:  "",
			ruleRoutePrefix: "/legacy",
			wantMatch:       true,
		},
		{
			name:            "no route or path attribute returns false",
			attrs:           map[string]string{"other.attr": "value"},
			ruleRouteExact:  "/api",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			got := matchHttpRoute(span, tt.ruleRouteExact, tt.ruleRoutePrefix)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}
