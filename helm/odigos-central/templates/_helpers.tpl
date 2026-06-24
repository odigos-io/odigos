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

{{/* Cloud connectors are installed only when the install secret exists AND the feature is enabled. */}}
{{- define "connectors.enabled" -}}
  {{- if and (include "odigos.secretExists" .) .Values.cloudConnectors.enabled -}}
true
  {{- end -}}
{{- end -}}

{{/* PostgreSQL DSN for the connector store, assembled from values. */}}
{{- define "connectors.postgresDSN" -}}
postgres://{{ .Values.cloudConnectors.postgres.username }}:{{ .Values.cloudConnectors.postgres.password }}@odigos-connector-postgres:{{ .Values.cloudConnectors.postgres.port }}/{{ .Values.cloudConnectors.postgres.database }}?sslmode=disable
{{- end -}}
