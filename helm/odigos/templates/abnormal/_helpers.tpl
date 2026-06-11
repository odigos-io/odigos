{{/*
Whether the bundled ClickHouse should be deployed: Abnormal enabled AND storage backend is clickhouse.
*/}}
{{- define "abnormal.clickhouse.enabled" -}}
{{- and .Values.abnormal.enabled (eq .Values.abnormal.storage "clickhouse") -}}
{{- end -}}

{{/*
Default ClickHouse image tag (current LTS line). Overridable via abnormal.clickhouse.tag.
*/}}
{{- define "abnormal.clickhouse.defaultTag" -}}26.3{{- end -}}

{{/*
Resolve the ClickHouse image.
- If abnormal.clickhouse.image is set, use it verbatim (supports arbitrary/air-gapped registries).
- Otherwise build `<imagePrefix>/odigos-clickhouse:<tag>` via the shared helper, so internal
  registries configured through imagePrefix are honored automatically.
*/}}
{{- define "abnormal.clickhouse.image" -}}
{{- $ch := .Values.abnormal.clickhouse -}}
{{- if $ch.image -}}
{{- $ch.image -}}
{{- else -}}
{{- $tag := $ch.tag | default (include "abnormal.clickhouse.defaultTag" .) -}}
{{- include "utils.imageName" (dict "Values" .Values "Component" "clickhouse" "Tag" $tag) -}}
{{- end -}}
{{- end -}}

{{/*
In-cluster DNS name of the ClickHouse client Service.
*/}}
{{- define "abnormal.clickhouse.host" -}}
odigos-clickhouse.{{ .Release.Namespace }}.svc.cluster.local
{{- end -}}

{{/*
Whether the chart owns (generates/holds) the ClickHouse password.
False when the user brings their own Secret via auth.existingSecret.
*/}}
{{- define "abnormal.clickhouse.chartOwnsPassword" -}}
{{- if .Values.abnormal.clickhouse.auth.existingSecret -}}false{{- else -}}true{{- end -}}
{{- end -}}

{{/*
Name of the Secret the StatefulSet reads the password from.
*/}}
{{- define "abnormal.clickhouse.passwordSecretName" -}}
{{- with .Values.abnormal.clickhouse.auth.existingSecret -}}{{ . }}{{- else -}}odigos-clickhouse-connection{{- end -}}
{{- end -}}

{{/*
Key within the password Secret that holds the plaintext password.
*/}}
{{- define "abnormal.clickhouse.passwordSecretKey" -}}
{{- with .Values.abnormal.clickhouse.auth.existingSecret -}}
{{- $.Values.abnormal.clickhouse.auth.existingSecretKey | default "password" -}}
{{- else -}}CLICKHOUSE_PASSWORD{{- end -}}
{{- end -}}

{{/*
Resolve the chart-owned ClickHouse password, stable across upgrades:
  1. auth.password                          -> explicit value from values (GitOps-friendly)
  2. existing connection Secret value       -> preserve generated password across `helm upgrade`
  3. freshly generated random
Only evaluated when the chart owns the password (no auth.existingSecret).
*/}}
{{- define "abnormal.clickhouse.resolvedPassword" -}}
{{- $auth := .Values.abnormal.clickhouse.auth -}}
{{- if $auth.password -}}
{{- $auth.password -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace "odigos-clickhouse-connection" -}}
{{- if and $existing $existing.data (index $existing.data "CLICKHOUSE_PASSWORD") -}}
{{- index $existing.data "CLICKHOUSE_PASSWORD" | b64dec -}}
{{- else -}}
{{- randAlphaNum 32 -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Common labels for the ClickHouse resources.
*/}}
{{- define "abnormal.clickhouse.labels" -}}
app.kubernetes.io/name: odigos-clickhouse
app.kubernetes.io/part-of: odigos-abnormal
odigos.io/system-object: "true"
{{- end -}}

{{/*
Selector labels for the ClickHouse pods.
*/}}
{{- define "abnormal.clickhouse.selectorLabels" -}}
app.kubernetes.io/name: odigos-clickhouse
{{- end -}}
