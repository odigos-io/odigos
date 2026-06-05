{{- define "utils.imagePrefix" -}}
{{- $enterprise := .Enterprise | default true -}}
{{- $defaultRegistry := "enterprise-registry.odigos.io" -}}
{{- if not $enterprise -}}
{{- $defaultRegistry = "registry.odigos.io" -}}
{{- end -}}
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
{{- if and (include "odigos.onPremToken" .) (include "odigos.usesDefaultEnterpriseRegistry" .) -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.usesDefaultEnterpriseRegistry" -}}
{{- eq (trimSuffix "/" (include "utils.imagePrefix" (dict "Values" .Values "Enterprise" true))) "enterprise-registry.odigos.io" -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryPullSecretToMount" -}}
{{- if and .Enterprise (include "odigos.hasEnterpriseRegistryPullSecret" .) -}}
{{- include "odigos.enterpriseRegistryPullSecretName" . -}}
{{- end -}}
{{- end -}}

{{- define "odigos.validateEnterpriseRegistryPullSecrets" -}}
{{- if and (include "odigos.secretExists" .) (include "odigos.usesDefaultEnterpriseRegistry" .) (not (include "odigos.onPremToken" .)) -}}
{{- fail "Enterprise images pull from enterprise-registry.odigos.io but no on-prem token is available to the chart. Set onPremToken or ensure the odigos-central secret exists in the release namespace before install/upgrade." -}}
{{- end -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryDockerConfigJson" -}}
{{- $token := include "odigos.onPremToken" . -}}
{{- $registry := trimSuffix "/" (include "utils.imagePrefix" (dict "Values" .Values "Enterprise" true)) -}}
{{- $auth := printf "odigos:%s" $token | b64enc -}}
{{- dict "auths" (dict $registry (dict "username" "odigos" "password" $token "auth" $auth)) | toJson -}}
{{- end -}}

{{- define "utils.imageName" -}}
{{- include "utils.imagePrefix" (dict "Values" $.Values "Enterprise" true) -}}
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

{{/* Render imagePullSecrets. Set Enterprise=true on workloads that pull enterprise images. */}}
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
