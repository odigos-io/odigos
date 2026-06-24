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
{{- if and (include "odigos.onPremToken" .) (include "odigos.usesOdigosRegistry" .) -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.usesOdigosRegistry" -}}
{{- if or $.Values.imagePrefix $.Values.openshift.enabled -}}
{{- else -}}
true
{{- end -}}
{{- end -}}

{{- define "odigos.enterpriseRegistryPullSecretToMount" -}}
{{- if include "odigos.hasEnterpriseRegistryPullSecret" . -}}
{{- include "odigos.enterpriseRegistryPullSecretName" . -}}
{{- end -}}
{{- end -}}

{{- define "odigos.validateEnterpriseRegistryPullSecrets" -}}
{{- if and (include "odigos.secretExists" .) (include "odigos.usesOdigosRegistry" .) (not (include "odigos.onPremToken" .)) -}}
{{- fail "Odigos images pull from registry.odigos.io but no on-prem token is available to the chart. Set onPremToken or ensure the odigos-central secret exists in the release namespace before install/upgrade." -}}
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
