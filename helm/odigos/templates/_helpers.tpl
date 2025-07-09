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

{{/* Define shared resource defaults */}}
{{- define "odigos.defaults.resources" -}}
{{- dict
  "cpu"    "500m"
  "memory" "512Mi"
-}}
{{- end }}

{{- define "odigos.odiglet.resolvedResources" -}}
{{- $defaults := include "odigos.defaults.resources" . | fromYaml -}}
{{- $resources := .Values.odiglet.resources | default dict | deepCopy -}}
{{- $requests := get $resources "requests" | default dict -}}
{{- $limits := get $resources "limits" | default dict -}}

{{- if and (empty $limits) (not (empty $requests)) -}}
  {{- $_ := set $resources "limits" $requests -}}
{{- else if and (empty $requests) (not (empty $limits)) -}}
  {{- $_ := set $resources "requests" $limits -}}
{{- else if and (empty $limits) (empty $requests) -}}
  {{- $_ := set $resources "limits" $defaults -}}
  {{- $_ := set $resources "requests" $defaults -}}
{{- end }}
{{- toYaml $resources -}}
{{- end }}


{{- define "odigos.odiglet.gomemlimitFromLimit" -}}
{{- $resources := include "odigos.odiglet.resolvedResources" . | fromYaml -}}
{{- $limits := get $resources "limits" -}}

{{- $raw := get $limits "memory" | trim -}}
{{- $number := regexFind "^[0-9]+" $raw -}}
{{- $unit := regexFind "[a-zA-Z]+$" $raw -}}

{{- if and $number $unit }}
  {{- $num := int $number -}}
  {{- $val := div (mul $num 80) 100 -}}

  {{- /*
    GOMEMLIMIT must use units like "MiB" or "GiB", while Kubernetes memory limits use "Mi", "Gi", etc.
    Since we derive GOMEMLIMIT from the memory limit, we append "B" to the unit.
  */ -}}
  {{- printf "%d%sB" $val $unit -}}
{{- else }}
  {{- fail (printf "Invalid memory limit format for GOMEMLIMIT: %q" $raw) -}}
{{- end }}
{{- end }}