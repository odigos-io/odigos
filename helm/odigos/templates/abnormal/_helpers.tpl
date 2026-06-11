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
Name of the Secret holding the ClickHouse password.
Uses the user-provided existingSecret when set, otherwise the chart-managed Secret.
*/}}
{{- define "abnormal.clickhouse.secretName" -}}
{{- $auth := .Values.abnormal.clickhouse.auth -}}
{{- if $auth.existingSecret -}}
{{- $auth.existingSecret -}}
{{- else -}}
odigos-clickhouse-auth
{{- end -}}
{{- end -}}

{{/*
Key within the password Secret that holds the plaintext password.
*/}}
{{- define "abnormal.clickhouse.secretKey" -}}
{{- $auth := .Values.abnormal.clickhouse.auth -}}
{{- if $auth.existingSecret -}}
{{- $auth.existingSecretKey | default "password" -}}
{{- else -}}
password
{{- end -}}
{{- end -}}

{{/*
Resolve the ClickHouse password for the chart-managed Secret, stable across upgrades:
explicit auth.password > existing chart-managed Secret value > freshly generated random.
(Only used when abnormal.clickhouse.auth.existingSecret is NOT set.)
*/}}
{{- define "abnormal.clickhouse.password" -}}
{{- $auth := .Values.abnormal.clickhouse.auth -}}
{{- if $auth.password -}}
{{- $auth.password -}}
{{- else -}}
{{- $existing := lookup "v1" "Secret" .Release.Namespace "odigos-clickhouse-auth" -}}
{{- if and $existing $existing.data (index $existing.data "password") -}}
{{- index $existing.data "password" | b64dec -}}
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
