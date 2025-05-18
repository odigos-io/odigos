{{- define "utils.imagePrefix" -}}
{{- $defaultRegistry := "registry.odigos.io" -}}
{{- $redHatRegistry := "registry.connect.redhat.com/odigos" -}}
{{- if $.Values.imagePrefix -}}
    {{- $.Values.imagePrefix -}}
{{- else -}}
    {{- if $.Values.openshift.enabled -}}
        {{- $redHatRegistry -}}
    {{- else -}}
        {{- $defaultRegistry -}}
    {{- end -}}
{{- end -}}
{{- end -}}

{{- define "utils.imageName" -}}
{{- printf "%s/odigos-%s%s:%s" (include "utils.imagePrefix" .) .Component (ternary "-ubi9" "" $.Values.openshift.enabled) .Tag }}
{{- end -}}
{{/*
Returns "true" if any userInstrumentationEnvs.language is enabled or has env vars
*/}}
{{- define "utils.shouldRenderUserInstrumentationEnvs" -}}
  {{- $languages := .Values.userInstrumentationEnvs.languages | default dict }}
  {{- $shouldRender := false }}
  {{- range $lang, $config := $languages }}
    {{- if or $config.enabled $config.env }}
      {{- $shouldRender = true }}
    {{- end }}
  {{- end }}
  {{- print $shouldRender }}
{{- end }}


