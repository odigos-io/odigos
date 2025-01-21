{{ define "packages" -}}

{{- range $idx, $val := .packages -}}
{{/* Special handling for kubeconfig */}}
{{- if eq .Title "kubeconfig (v1)" -}}
---
title: {{ .Title }}
content_type: tool-reference
package: v1
auto_generated: true
---
{{- else -}}
  {{- if and .IsMain (ne .GroupName "") -}}
---
title: {{ .Title }}
content_type: tool-reference
package: {{ .DisplayName }}
auto_generated: true
---
{{ .GetComment -}}
  {{- end -}}
{{- end -}}
{{- end }}

## Resource Types 

{{ range .packages -}}
  {{- range .VisibleTypes -}}
    {{- if .IsExported }}
- [{{ .DisplayName }}]({{ .Link }})
    {{- end -}}
  {{- end -}}
{{- end -}}

{{ range .packages }}
  {{ if ne .GroupName "" -}}
    {{/* For package with a group name, list all type definitions in it. */}}
    {{- range .VisibleTypes }}
      {{- if or .Referenced .IsExported -}}
{{ template "type" . }}
      {{- end -}}
    {{ end }}
  {{ else }}
    {{/* For package w/o group name, list only types referenced. */}}
    {{ $pkgTitle := .Title }}
    {{- range .VisibleTypes -}}
      {{- if or .Referenced (eq $pkgTitle "kubeconfig (v1)") -}}
{{ template "type" . }}
      {{- end -}}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}