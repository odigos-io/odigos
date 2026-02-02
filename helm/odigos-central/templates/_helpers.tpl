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
-certified
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

{{/*
  Calculate GOMEMLIMIT from Kubernetes resource limits.
  Takes 80% of the memory limit to leave headroom for non-heap memory.
*/}}
{{- define "odigos.gomemlimitFromResources" -}}
{{- $resources := .Resources | default dict -}}
{{- $limits := get $resources "limits" -}}
{{- $requests := get $resources "requests" -}}

{{- $raw := (get $limits "memory") | default (get $requests "memory") -}}
{{- $number := regexFind "^[0-9]+" $raw -}}
{{- $unit := regexFind "[a-zA-Z]+$" $raw -}}

{{- if and $number $unit }}
  {{- $num := int $number -}}
  {{- $val := div (mul $num 80) 100 -}}
  {{- /*
     GOMEMLIMIT requires units like "MiB" or "GiB", whereas Kubernetes uses "Mi" or "Gi".
     To ensure Go runtime parses the value correctly, we must always append "B" to the unit.
*/ -}}
  {{- printf "%d%sB" $val $unit -}}
{{- else }}
   {{- fail (printf "Invalid memory limit format for GOMEMLIMIT: %q" $raw) -}}
{{- end }}
{{- end }}
