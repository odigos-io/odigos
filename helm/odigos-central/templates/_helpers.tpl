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
{{- $prefix := include "utils.imagePrefix" (dict "Values" $.Values) -}}
{{- printf "%s/odigos-enterprise-%s%s:%s" $prefix .Component (ternary "-rhel-certified" "" $.Values.openshift.enabled) .Tag -}}
{{- end -}}

{{- define "odigos.onPremToken" -}}
{{- if .Values.onPremToken -}}
{{- .Values.onPremToken -}}
{{- else -}}
{{- $sec := lookup "v1" "Secret" .Release.Namespace "odigos-central" -}}
{{- if $sec -}}
{{- if index $sec.data "ODIGOS_ONPREM_TOKEN" -}}
{{- index $sec.data "ODIGOS_ONPREM_TOKEN" | b64dec -}}
{{- else if index $sec.data "odigos-onprem-token" -}}
{{- index $sec.data "odigos-onprem-token" | b64dec -}}
{{- end -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryPullSecretName" -}}
odigos-enterprise-registry
{{- end -}}

{{- define "odigos.hasEnterpriseRegistryPullSecret" -}}
{{- if not (include "odigos.usesOdigosRegistry" .) -}}
{{- else if and (.Values.externalOnpremPullSecret | default false) (include "odigos.secretExists" .) -}}
true
{{- else if include "odigos.onPremToken" . -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.createEnterpriseRegistryPullSecret" -}}
{{- if and (include "odigos.usesOdigosRegistry" .) (include "odigos.onPremToken" .) (not (.Values.externalOnpremPullSecret | default false)) -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.usesOdigosRegistry" -}}
{{- if eq (include "utils.imagePrefix" (dict "Values" .Values)) "registry.odigos.io" -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryPullSecretToMount" -}}
{{- if include "odigos.hasEnterpriseRegistryPullSecret" . -}}
{{- include "odigos.enterpriseRegistryPullSecretName" . -}}
{{- end -}}
{{- end -}}

{{- define "odigos.validateEnterpriseRegistryPullSecrets" -}}
{{- if and (include "odigos.secretExists" .) (include "odigos.usesOdigosRegistry" .) (not (include "odigos.onPremToken" .)) (not (.Values.externalOnpremPullSecret | default false)) -}}
{{- fail "Odigos images pull from registry.odigos.io but no on-prem token is available to the chart. Set onPremToken, set externalOnpremPullSecret to true when providing odigos-enterprise-registry externally, or ensure the odigos-central secret exists in the release namespace before install/upgrade." -}}
{{- end -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryDockerConfigJson" -}}
{{- $token := include "odigos.onPremToken" . -}}
{{- $auth := printf "odigos:%s" $token | b64enc -}}
{{- dict "auths" (dict "registry.odigos.io" (dict "username" "odigos" "password" $token "auth" $auth)) | toJson -}}
{{- end -}}

{{- define "odigos.secretExists" -}}
  {{- $sec   := lookup "v1" "Secret" .Release.Namespace "odigos-central" -}}
  {{- $token := default "" .Values.onPremToken -}}
  {{- $externalSecret := .Values.externalOnpremTokenSecret | default false -}}
  {{- if or $sec (ne $token "") $externalSecret -}}
true
  {{- end -}}
{{- end -}}

{{/* Render imagePullSecrets, including the on-prem registry pull secret when onPremToken is set. */}}
{{- define "odigos.renderPullSecrets" -}}
{{- $enterpriseSecret := include "odigos.enterpriseRegistryPullSecretToMount" . -}}
{{- if or .Values.imagePullSecrets $enterpriseSecret }}
imagePullSecrets:
{{- range .Values.imagePullSecrets }}
  - name: {{ . | quote }}
{{- end }}
{{- if $enterpriseSecret }}
  - name: {{ $enterpriseSecret | quote }}
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
