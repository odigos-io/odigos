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


{{- define "odigos.odiglet.gomemlimitFromLimit" -}}

{{- $resources := .Values.odiglet.resources | default dict -}}
{{- $limits := get $resources "limits" | default dict -}}
{{- $requests := get $resources "requests" | default dict -}}

{{- $memFromLimits := get $limits "memory" -}}
{{- $memFromRequests := get $requests "memory" -}}

{{/* Use limits.memory if set, otherwise fallback to requests.memory, or default to 512Mi */}}
{{- $raw := $memFromLimits | default $memFromRequests | default "512Mi" | trim -}}

{{- $number := regexFind "^[0-9]+" $raw -}}
{{- $unit := regexFind "[a-zA-Z]+$" $raw -}}

{{- if and $number $unit }}
  {{- $num := int $number -}}
  {{- $val := div (mul $num 80) 100 -}}
  {{/*
  GOMEMLIMIT must use units like "MiB" or "GiB", while Kubernetes memory limits use "Mi", "Gi", etc.
  Since we derive GOMEMLIMIT from the memory limit, we append "B" to the unit if it's not already present.
  */}}
  {{- printf "%d%sB" $val $unit -}}
{{- else }}
  {{/* Fallback to a default value if parsing fails */}}
  {{- "409MiB" -}}
{{- end }}
{{- end }}