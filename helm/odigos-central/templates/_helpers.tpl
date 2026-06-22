{{- define "utils.imageName" -}}
{{- $defaultRegistry := "p0xd21zf5r.registry.depot.dev/odigos" -}}
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
odigos-enterprise-{{- .Component -}}
{{- if $.Values.openshift.enabled -}}
-rhel-certified
{{- end -}}
:
{{- .Tag -}}
{{- end -}}

{{- define "odigos.secretExists" -}}
  {{- $sec   := lookup "v1" "Secret" .Release.Namespace "odigos-central" -}}
  {{- $token := default "" .Values.onPremToken -}}
  {{- $externalSecret := .Values.externalOnpremTokenSecret | default false -}}
  {{- if or $sec (ne $token "") $externalSecret -}}
true
  {{- end -}}
{{- end -}}

{{/* Render imagePullSecrets in K8s shape from a list of strings */}}
{{- define "odigos.renderPullSecrets" -}}
{{- if .Values.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.imagePullSecrets }}
  - name: {{ . | quote }}
{{- end }}
{{- end }}
{{- end }}
