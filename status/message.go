package status

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// WithMessageTemplate returns a copy of r with Template set from Message when
// Message contains template syntax. Panics on parse errors so invalid status
// YAML fails at package init rather than at runtime.
func WithMessageTemplate(r Reason) Reason {
	tmpl, err := parseMessageTemplate(r.Message)
	if err != nil {
		// this is static and run once on startup, so we can panic here
		panic(fmt.Sprintf("status reason %q: %v", r.Name, err))
	}
	r.Template = tmpl
	return r
}

func parseMessageTemplate(message string) (*template.Template, error) {
	if !strings.Contains(message, "{{") {
		return nil, nil
	}
	// set missingkey=error; the default replaces missing keys with "<no value>"
	return template.New("message").Option("missingkey=error").Parse(message)
}

// RenderMessage renders a reason's Message using its pre-parsed Template and the
// given parameters. Reasons without a Template return Message unchanged.
func RenderMessage(r Reason, params any) (string, error) {
	if r.Template == nil {
		return r.Message, nil
	}

	var buf bytes.Buffer
	if err := r.Template.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("render status message template: %w", err)
	}
	return buf.String(), nil
}
