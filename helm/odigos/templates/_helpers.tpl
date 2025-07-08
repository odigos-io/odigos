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

{{- define "odigos.secretExists" -}}
  {{- $sec   := lookup "v1" "Secret" .Release.Namespace "odigos-pro" -}}
  {{- $token := default "" .Values.onPremToken -}}
  {{- if or $sec (ne $token "") -}}
true
  {{- end -}}
{{- end -}}


{{/*
  Return cleaned Kubernetes version, keeping leading 'v', removing vendor suffix like -eks-...
  */}}
  {{- define "utils.cleanKubeVersion" -}}
  {{- regexReplaceAll "-.*" .Capabilities.KubeVersion.Version "" -}}
  {{- end }}

{{- define "odigos.odiglet.resources" -}}
{{- $defaults := dict
  "cpu"    "500m"
  "memory" "512Mi"
-}}

{{- $resources := .Values.odiglet.resources | default dict -}}
{{- $requests := get $resources "requests" | default dict -}}
{{- $limits := get $resources "limits" | default dict -}}
{{- if and (empty $limits) (not (empty $requests)) -}}
  {{- $_ := set $resources "limits" $requests -}}
{{- end }}
{{- if and (empty $limits) (empty $requests) -}}
  {{- $_ := set $resources "limits" $defaults -}}
  {{- $_ := set $resources "requests" $defaults -}}
{{- end }}
{{- toYaml $resources | indent 12 }}
{{- end }}


{{- define "odigos.gomemlimitFromLimits" -}}
{{- $resources := .Values.odiglet.resources | default dict -}}
{{- $limits := get $resources "limits" | default dict -}}
{{- $requests := get $resources "requests" | default dict -}}

{{- $raw := get $limits "memory" | default (get $requests "memory" | default "512Mi") | trim -}}

{{- $number := regexFind "^[0-9]+" $raw -}}
{{- $unit := regexFind "[a-zA-Z]+$" $raw | default "Mi" -}}

{{- if and $number $unit }}
  {{- $num := int $number -}}
  {{- $val := divf (mul $num 80.0) 100.0 -}}
  {{- if hasSuffix $unit "B" -}}
    {{- printf "%.1f%s" $val $unit -}}
  {{- else -}}
    {{- printf "%.1f%sB" $val $unit -}}
  {{- end -}}
{{- else }}
  {{- fail (printf "Invalid memory format for limit: %q" $raw) -}}
{{- end }}
{{- end }}
