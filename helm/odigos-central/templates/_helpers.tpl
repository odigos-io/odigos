{{- define "utils.imageName" -}}
{{- $defaultRegistry := "registry.odigos.io" -}}
{{- $redHatRegistry := "registry.connect.redhat.com/odigos" -}}
{{- if $.Values.imagePrefix -}}
    {{- $.Values.imagePrefix -}}/
{{- else -}}
    {{- if $.Values.openshift.enabled -}}
        {{- $redHatRegistry -}}/
    {{- else -}}
        {{- $defaultRegistry -}}/
    {{- end -}}
{{- end -}}
{{- if or (eq .Component "central-backend") (eq .Component "central-ui") -}}
odigos-enterprise-{{- .Component -}}
{{- else -}}
odigos-{{- .Component -}}
{{- end -}}
{{- if $.Values.openshift.enabled -}}
-ubi9
{{- end -}}
:
{{- .Tag -}}
{{- end -}}
