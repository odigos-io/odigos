{{/*
    _sizing-helpers.tpl
    Hybrid sizing helpers:
    - Defaults come from .Values.ResourceSizePreset (size_s | size_m | size_l)
    - Users may override specific CPU/Memory/replica knobs safely
    - Memory-limiter trio (limit/spike/go) is auto-derived from limit unless all three are set explicitly
  */}}

  {{/* ------------------------------------------------------------------
       0) Validate and resolve ResourceSizePreset
  ------------------------------------------------------------------ */}}

  {{- define "collector.validateSizing" -}}
  {{- $s := .Values.ResourceSizePreset | default "size_m" -}}
  {{- if not (has $s (list "size_s" "size_m" "size_l")) -}}
    {{- fail (printf "Invalid ResourceSizePreset=%q. Valid: size_s, size_m, size_l" $s) -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.sizing.resolve" -}}
  {{- include "collector.validateSizing" . -}}
  {{- .Values.ResourceSizePreset | default "size_m" -}}
  {{- end -}}


  {{/* ------------------------------------------------------------------
       1) Sizing tables (CPU/Memory/Replicas)
       Keys are simple numbers (MiB for mem, m for CPU)
  ------------------------------------------------------------------ */}}

  {{/*
    collector.sizingDefaults
    -------------------------
    Returns a YAML block of default sizing values (CPU/Memory/Replicas) for
    collectorGateway and collectorNode based on .Values.ResourceSizePreset.
    - ResourceSizePreset may be: size_s, size_m, size_l (default = size_m)
    - Numbers are emitted as integers (not strings) so fromYaml works properly
    - | trim is used on ResourceSizePreset to avoid newline/space mismatch
  */}}

  {{- define "collector.sizingDefaults" -}}
  {{- $s := (include "collector.sizing.resolve" . | trim) -}}
  {{- if eq $s "size_s" }}
  gatewayMinReplicas: 1
  gatewayMaxReplicas: 5
  gatewayMemoryRequest: 300
  gatewayMemoryLimit: 300
  gatewayCPURequest: 150
  gatewayCPULimit: 300
  nodeMemoryRequest: 150
  nodeMemoryLimit: 300
  nodeCPURequest: 150
  nodeCPULimit: 300
  {{- else if eq $s "size_l" }}
  gatewayMinReplicas: 3
  gatewayMaxReplicas: 12
  gatewayMemoryRequest: 850
  gatewayMemoryLimit: 850
  gatewayCPURequest: 750
  gatewayCPULimit: 1250
  nodeMemoryRequest: 500
  nodeMemoryLimit: 750
  nodeCPURequest: 500
  nodeCPULimit: 750
  {{- else }} {{/* default size_m */}}
  gatewayMinReplicas: 2
  gatewayMaxReplicas: 8
  gatewayMemoryRequest: 600
  gatewayMemoryLimit: 600
  gatewayCPURequest: 500
  gatewayCPULimit: 1000
  nodeMemoryRequest: 250
  nodeMemoryLimit: 500
  nodeCPURequest: 250
  nodeCPULimit: 500
  {{- end }}
  {{- end }}


  {{ include "collector.sizingDefaults" . }}

  {{/* Derive limiter trio from a given memory LIMIT (MiB) */}}
  {{- define "collector._limiterFromLimit" -}}
  {{- $limit := (index . "limit") | int -}}
  {{- $hard := sub $limit 50 -}}
  {{- $spike := div (mul $hard 20) 100 -}}
  {{- $go := div (mul $hard 80) 100 -}}
  limitMiB: {{ $hard }}
  spikeMiB: {{ $spike }}
  goMiB: {{ $go }}
  {{- end -}}


  {{/* ------------------------------------------------------------------
       2) Gateway: effective CPU/Memory with mirroring rules
  ------------------------------------------------------------------ */}}

  {{- define "collector.gateway.memoryRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorGateway "requestMemoryMiB" -}}
  {{- .Values.collectorGateway.requestMemoryMiB -}}
  {{- else if hasKey .Values.collectorGateway "limitMemoryMiB" -}}
  {{- .Values.collectorGateway.limitMemoryMiB -}}
  {{- else -}}
  {{- $d.gatewayMemoryRequest -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.gateway.memoryLimit" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorGateway "limitMemoryMiB" -}}
  {{- .Values.collectorGateway.limitMemoryMiB -}}
  {{- else if hasKey .Values.collectorGateway "requestMemoryMiB" -}}
  {{- .Values.collectorGateway.requestMemoryMiB -}}
  {{- else -}}
  {{- $d.gatewayMemoryLimit -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.gateway.cpuRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorGateway "requestCPUm" -}}
  {{- .Values.collectorGateway.requestCPUm -}}
  {{- else if hasKey .Values.collectorGateway "limitCPUm" -}}
  {{- .Values.collectorGateway.limitCPUm -}}
  {{- else -}}
  {{- $d.gatewayCPURequest -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.gateway.cpuLimit" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorGateway "limitCPUm" -}}
  {{- .Values.collectorGateway.limitCPUm -}}
  {{- else if hasKey .Values.collectorGateway "requestCPUm" -}}
  {{- .Values.collectorGateway.requestCPUm -}}
  {{- else -}}
  {{- $d.gatewayCPULimit -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.gateway.minReplicas" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- $min := (get .Values.collectorGateway "minReplicas" | default $d.gatewayMinReplicas) | int -}}
  {{- $max := (get .Values.collectorGateway "maxReplicas" | default $d.gatewayMaxReplicas) | int -}}
  {{- if gt $min $max -}}
    {{- fail (printf "collectorGateway.minReplicas (%d) must be <= collectorGateway.maxReplicas (%d)" $min $max) -}}
  {{- end -}}
  {{- $min -}}
{{- end -}}

  {{- define "collector.gateway.maxReplicas" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- (get .Values.collectorGateway "maxReplicas" | default $d.gatewayMaxReplicas) | int -}}
  {{- end -}}



  {{/* ------------------------------------------------------------------
       3) Node: effective CPU/Memory with mirroring rules
  ------------------------------------------------------------------ */}}

  {{- define "collector.node.memoryRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "requestMemoryMiB" -}}
  {{- .Values.collectorNode.requestMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "memoryRequest" -}}
  {{/* Backward compatibility: support legacy field name "memoryRequest" */}}
  {{- .Values.collectorNode.memoryRequest -}}
  {{- else if hasKey .Values.collectorNode "limitMemoryMiB" -}}
  {{- .Values.collectorNode.limitMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "memoryLimit" -}}
  {{/* Backward compatibility: support legacy field name "memoryLimit" for mirroring */}}
  {{- .Values.collectorNode.memoryLimit -}}
  {{- else -}}
  {{- $d.nodeMemoryRequest -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.memoryLimit" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "limitMemoryMiB" -}}
  {{- .Values.collectorNode.limitMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "memoryLimit" -}}
  {{/* Backward compatibility: support legacy field name "memoryLimit" */}}
  {{- .Values.collectorNode.memoryLimit -}}
  {{- else if hasKey .Values.collectorNode "requestMemoryMiB" -}}
  {{- .Values.collectorNode.requestMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "memoryRequest" -}}
  {{/* Backward compatibility: support legacy field name "memoryRequest" for mirroring */}}
  {{- .Values.collectorNode.memoryRequest -}}
  {{- else -}}
  {{- $d.nodeMemoryLimit -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.cpuRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "requestCPUm" -}}
  {{- .Values.collectorNode.requestCPUm -}}
  {{- else if hasKey .Values.collectorNode "cpuRequest" -}}
  {{/* Backward compatibility: support legacy field name "cpuRequest" */}}
  {{- .Values.collectorNode.cpuRequest -}}
  {{- else if hasKey .Values.collectorNode "limitCPUm" -}}
  {{- .Values.collectorNode.limitCPUm -}}
  {{- else -}}
  {{- $d.nodeCPURequest -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.cpuLimit" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "limitCPUm" -}}
  {{- .Values.collectorNode.limitCPUm -}}
  {{- else if hasKey .Values.collectorNode "requestCPUm" -}}
  {{- .Values.collectorNode.requestCPUm -}}
  {{- else if hasKey .Values.collectorNode "cpuRequest" -}}
  {{/* Backward compatibility: support legacy field name "cpuRequest" for mirroring */}}
  {{- .Values.collectorNode.cpuRequest -}}
  {{- else -}}
  {{- $d.nodeCPULimit -}}
  {{- end -}}
  {{- end -}}


  {{/* ------------------------------------------------------------------
       4) Memory-limiter trios (gateway/node)
       Rule: if any of the trio is set → require all three
       Else: derive all three from effective memory LIMIT
  ------------------------------------------------------------------ */}}

  {{/* Generic memoryLimiter helper */}}
  {{- define "collector._memoryLimiter" -}}
    {{- $vals := .vals | default dict -}}
    {{- $limitTpl := .limitTpl -}}
    {{- $who := .who -}}
    {{- $ctx := .ctx -}}

    {{- $hasLimit := hasKey $vals "memoryLimiterLimitMiB" -}}
    {{- $hasSpike := hasKey $vals "memoryLimiterSpikeLimitMiB" -}}
    {{- $hasGo    := hasKey $vals "goMemLimitMiB" -}}

    {{- if or $hasLimit $hasSpike $hasGo -}}
      {{- if not (and $hasLimit $hasSpike $hasGo) -}}
        {{- fail (printf "%s: if any of memoryLimiterLimitMiB/memoryLimiterSpikeLimitMiB/goMemLimitMiB is set, all three must be set" $who) -}}
      {{- end -}}
      {{- toYaml (dict "limitMiB" $vals.memoryLimiterLimitMiB "spikeMiB" $vals.memoryLimiterSpikeLimitMiB "goMiB" $vals.goMemLimitMiB) -}}
    {{- else -}}
      {{- $memLimit := include $limitTpl $ctx | int -}}
      {{- $hard := sub $memLimit 50 -}}
      {{- $spike := div (mul $hard 20) 100 -}}
      {{- $go := div (mul $hard 80) 100 -}}
      {{- toYaml (dict "limitMiB" $hard "spikeMiB" $spike "goMiB" $go) -}}
    {{- end -}}
  {{- end -}}


    {{/* Gateway entrypoint */}}
    {{- define "collector.gateway.memoryLimiter" -}}
      {{- include "collector._memoryLimiter" (dict "vals" .Values.collectorGateway "limitTpl" "collector.gateway.memoryLimit" "who" "collectorGateway" "ctx" .) -}}
    {{- end -}}

    {{/* Node entrypoint */}}
    {{- define "collector.node.memoryLimiter" -}}
      {{- include "collector._memoryLimiter" (dict "vals" .Values.collectorNode "limitTpl" "collector.node.memoryLimit" "who" "collectorNode" "ctx" .) -}}
    {{- end -}}
