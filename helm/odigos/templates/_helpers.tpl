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
{{- /* Check for component-specific image override in .Values.images.<component> */ -}}
{{- $images := $.Values.images | default dict -}}
{{- $componentImage := get $images .Component -}}
{{- if $componentImage -}}
  {{- $componentImage -}}
{{- else -}}
  {{- $certified := $.Values.openshift.enabled }}
  {{- if hasKey $.Values.openshift "certifiedImageTags" }}
    {{- $certified = $.Values.openshift.certifiedImageTags }}
  {{- end }}
  {{- printf "%s/odigos-%s%s:%s" (include "utils.imagePrefix" .) .Component (ternary "-rhel-certified" "" (and $.Values.openshift.enabled $certified)) .Tag }}
{{- end -}}
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
  {{- $externalSecret := .Values.externalOnpremTokenSecret | default false -}}
  {{- if or $sec (ne $token "") $externalSecret -}}
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


{{- define "odigos.memoryQuantityMiB" -}}
{{- $raw := . -}}
{{- $number := regexFind "^[0-9]+" $raw -}}
{{- $unit := regexFind "[a-zA-Z]+$" $raw | lower -}}
{{- if and $number $unit }}
  {{- $num := int $number -}}
  {{- if eq $unit "ki" -}}
    {{- div $num 1024 -}}
  {{- else if eq $unit "mi" -}}
    {{- $num -}}
  {{- else if eq $unit "gi" -}}
    {{- mul $num 1024 -}}
  {{- else if eq $unit "ti" -}}
    {{- mul $num 1048576 -}}
  {{- else if eq $unit "k" -}}
    {{- div (mul $num 1000) 1048576 -}}
  {{- else if eq $unit "m" -}}
    {{- div (mul $num 1000000) 1048576 -}}
  {{- else if eq $unit "g" -}}
    {{- div (mul $num 1000000000) 1048576 -}}
  {{- else if eq $unit "t" -}}
    {{- div (mul $num 1000000000000) 1048576 -}}
  {{- else -}}
    {{- fail (printf "Invalid memory unit for GOMEMLIMIT: %q" $raw) -}}
  {{- end -}}
{{- else }}
  {{- fail (printf "Invalid memory limit format for GOMEMLIMIT: %q" $raw) -}}
{{- end }}
{{- end }}

{{- define "odigos.gomemlimitFromResources" -}}
{{- $resources := .Resources | default dict -}}
{{- $limits := get $resources "limits" -}}
{{- $requests := get $resources "requests" -}}

{{- $raw := (get $limits "memory") | default (get $requests "memory") -}}
{{- $limitMiB := int (include "odigos.memoryQuantityMiB" $raw) -}}
{{- $valMiB := div (mul $limitMiB 80) 100 -}}
{{- printf "%dMiB" $valMiB -}}
{{- end }}

{{/*
  Odiglet-specific GOMEMLIMIT: odiglet allocates significant memory for eBPF maps
  which live outside the Go heap. The Go runtime is unaware of this memory, so we
  use a lower GOMEMLIMIT percentage (default 60%) for memory limits >= 500Mi to
  leave enough headroom for eBPF allocations. Below 500Mi the standard 80% is used.
*/}}
{{- define "odigos.odiglet.gomemlimit" -}}
{{- $resources := include "odigos.odiglet.resolvedResources" . | fromYaml -}}
{{- $limits := get $resources "limits" -}}
{{- $requests := get $resources "requests" -}}

{{- $raw := (get $limits "memory") | default (get $requests "memory") -}}
{{- $limitMiB := int (include "odigos.memoryQuantityMiB" $raw) -}}
{{- $pct := 80 -}}
{{- if ge $limitMiB 500 -}}
  {{- $pct = int (default 60 .Values.odiglet.odiglet.goMemLimitPercentage) -}}
{{- end -}}
{{- $valMiB := div (mul $limitMiB $pct) 100 -}}
{{- printf "%dMiB" $valMiB -}}
{{- end }}

{{/* Returns "true" when UI requires write permissions (non-readonly UI or central backend set) */}}
{{- define "odigos.ui.requiresWritePermissions" -}}
  {{- if or (ne .Values.ui.uiMode "readonly") (ne .Values.centralProxy.centralBackendURL "") -}}
true
  {{- end -}}
{{- end -}}

{{- define "odigos.odiglet.sizing.resources" -}}
{{- $s := default "size_m" .Values.ResourceSizePreset -}}
{{- $sizes := dict
  "size_s" (dict "cpu" "150m" "memory" "300Mi")
  "size_m" (dict "cpu" "500m" "memory" "500Mi")
  "size_l" (dict "cpu" "750m" "memory" "750Mi")
  "size_xl" (dict "cpu" "1000m" "memory" "1024Mi")
-}}
{{- with (get $sizes $s) -}}
{{ toYaml . }}
{{- end -}}
{{- end }}

{{/* Comma-join pull secret names for CLI args */}}
{{- define "odigos.joinPullSecrets" -}}
{{- if .Values.imagePullSecrets -}}
{{- join "," .Values.imagePullSecrets -}}
{{- end -}}
{{- end }}

{{/* Render imagePullSecrets in K8s shape from a list of strings */}}
{{- define "odigos.renderPullSecrets" -}}
{{- if .Values.imagePullSecrets }}
imagePullSecrets:
{{- range .Values.imagePullSecrets }}
  - name: {{ . | quote }}
{{- end }}
{{- end }}
{{- end }}
