{{ define "members" }}
  {{/* . is a apiType */}}
  {{- range .GetMembers -}}
    {{/* . is a apiMember */}}
    {{- if not .Hidden }}
<tr>
<td>
<code>{{ .FieldName }}</code>{{- if not .IsOptional }} <B>[Required]</B>{{- end }}
</td>
<td>
{{/* Link for type reference */}}
      {{- with .GetType -}}
        {{- if .Link -}}
<a href="{{ .Link }}"><code>{{ .DisplayName }}</code></a>
        {{- else -}}
<code>{{ .DisplayName }}</code>
        {{- end -}}
      {{- end }}
</td>
<td>
   {{- if .IsInline -}}
(Members of <code>{{ .FieldName }}</code> are embedded into this type.)
   {{- end }}
   {{ if .GetComment -}}
   {{ .GetComment }}
   {{- else -}}
   <span class="text-muted">No description provided.</span>
   {{ end }}
   {{- if and (eq (.GetType.Name.Name) "ObjectMeta") -}}
Refer to the Kubernetes API documentation for the fields of the <code>metadata</code> field.
   {{ end -}}
</td>
</tr>
    {{- end }}
  {{- end }}
{{ end }}