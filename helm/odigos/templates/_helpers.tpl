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

{{- define "odigos.odiglet.resolvedResources" -}}
{{- $defaults := dict "cpu" "500m" "memory" "512Mi" -}}
{{- $resources := deepCopy (.Values.odiglet.resources | default dict) -}}

{{- $requests := get $resources "requests" | default dict -}}
{{- $limits := get $resources "limits" | default dict -}}

{{- if and (empty $limits) (not (empty $requests)) -}}
  {{- $_ := set $resources "limits" $requests -}}

{{- else if and (empty $requests) (not (empty $limits)) -}}
  {{- $_ := set $resources "requests" $limits -}}

{{- else if and (empty $limits) (empty $requests) -}}
  {{- $sizingYaml := include "odigos.odiglet.sizing.resources" . -}}
  {{- $sizing := $sizingYaml | fromYaml -}}
  {{- if $sizing }}
    {{- $_ := set $resources "limits" $sizing -}}
    {{- $_ := set $resources "requests" $sizing -}}
  {{- else }}
    {{- $_ := set $resources "limits" $defaults -}}
    {{- $_ := set $resources "requests" $defaults -}}
  {{- end }}
{{- end }}

{{- toYaml $resources -}}
{{- end }}


{{- define "odigos.odiglet.gomemlimitFromLimit" -}}
{{- $resources := include "odigos.odiglet.resolvedResources" . | fromYaml -}}
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

{{- define "odigos.odiglet.sizing.resources" -}}
{{- $profiles := .Values.profiles | default list -}}
{{- $profile := "" -}}
{{- range $profiles }}
  {{- if or (eq . "size_s") (eq . "size_m") (eq . "size_l") }}
    {{- $profile = . -}}
  {{- end }}
{{- end }}

{{- if eq $profile "size_s" }}
  {{- dict "cpu" "150m" "memory" "300Mi" | toYaml }}
{{- else if eq $profile "size_m" }}
  {{- dict "cpu" "500m" "memory" "500Mi" | toYaml }}
{{- else if eq $profile "size_l" }}
  {{- dict "cpu" "750m" "memory" "750Mi" | toYaml }}
{{- else }}
  {{- "" }}
{{- end }}
{{- end }}
