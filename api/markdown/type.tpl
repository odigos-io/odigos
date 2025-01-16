{{ define "type" }}

## `{{ .Name.Name }}` <a id="{{ .Anchor }}"></a>
    
{{ if eq .Kind "Alias" -}}
(Alias of `{{ .Underlying }}`)
{{ end }}

{{- with .References }}
**Appears in:**
{{ range . }}
{{ if or .Referenced .IsExported -}}
- [{{ .DisplayName }}]({{ .Link }})
{{ end -}}
{{- end -}}
{{- end }}

{{ if .GetComment -}}
{{ .GetComment }}
{{ end }}
{{ if .GetMembers -}}
<table class="table">
<thead><tr><th width="30%">Field</th><th>Type</th><th>Description</th></tr></thead>
<tbody>
    {{/* . is a apiType */}}
    {{- if .IsExported -}}
{{/* Add apiVersion and kind rows if deemed necessary */}}
<tr><td><code>apiVersion</code></td><td>string</td><td><code>{{- .APIGroup -}}</code></td></tr>
<tr><td><code>kind</code></td><td>string</td><td><code>{{- .Name.Name -}}</code></td></tr>
    {{ end -}}

{{/* The actual list of members is in the following template */}}
{{- template "members" . -}}
</tbody>
</table>
{{- end -}}
{{- end -}}