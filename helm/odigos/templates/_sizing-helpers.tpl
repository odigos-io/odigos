{{/*
    _sizing-helpers.tpl
    Hybrid sizing helpers:
    - Defaults come from .Values.sizingConfig (size_s | size_m | size_l)
    - Users may override specific CPU/Memory/replica knobs safely
    - Memory-limiter trio (limit/spike/go) is auto-derived from limit unless all three are set explicitly
  */}}

  {{/* ------------------------------------------------------------------ */
  /* 0) Validate and resolve sizingConfig                                 */
  /* ------------------------------------------------------------------ */}}

  {{- define "collector.validateSizing" -}}
  {{- $s := .Values.sizingConfig | default "size_m" -}}
  {{- if not (has $s (list "size_s" "size_m" "size_l")) -}}
    {{- fail (printf "Invalid sizingConfig=%q. Valid: size_s, size_m, size_l" $s) -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.sizing.resolve" -}}
  {{- include "collector.validateSizing" . -}}
  {{- .Values.sizingConfig | default "size_m" -}}
  {{- end -}}


  {{/* ------------------------------------------------------------------ */
  /* 1) Sizing tables (CPU/Memory/Replicas)                                */
  /*     Keys are simple numbers (MiB for mem, m for CPU)                  */
  /* ------------------------------------------------------------------ */}}

  {{- define "collector.sizingDefaults" -}}
  {{- $s := include "collector.sizing.resolve" . -}}
  {{- if eq $s "size_s" -}}
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
  {{- else if eq $s "size_l" -}}
  gatewayMinReplicas: 3
  gatewayMaxReplicas: 12
  gatewayMemoryRequest: 750
  gatewayMemoryLimit: 850
  gatewayCPURequest: 750
  gatewayCPULimit: 1250
  nodeMemoryRequest: 500
  nodeMemoryLimit: 750
  nodeCPURequest: 500
  nodeCPULimit: 750
  {{- else -}} {{/* size_m */}}
  gatewayMinReplicas: 2
  gatewayMaxReplicas: 8
  gatewayMemoryRequest: 500
  gatewayMemoryLimit: 600
  gatewayCPURequest: 500
  gatewayCPULimit: 1000
  nodeMemoryRequest: 250
  nodeMemoryLimit: 500
  nodeCPURequest: 250
  nodeCPULimit: 500
  {{- end -}}
  {{- end -}}

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

  {{/* Optional: a full defaults block that also includes limiter trio for gateway+node */}}
  {{- define "collector.sizingDefaults.full" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- $gwLimiter := include "collector._limiterFromLimit" (dict "limit" $d.gatewayMemoryLimit) | fromYaml -}}
  {{- $nodeLimiter := include "collector._limiterFromLimit" (dict "limit" $d.nodeMemoryLimit) | fromYaml -}}
  gatewayMinReplicas: {{ $d.gatewayMinReplicas }}
  gatewayMaxReplicas: {{ $d.gatewayMaxReplicas }}
  gatewayMemoryRequest: {{ $d.gatewayMemoryRequest }}
  gatewayMemoryLimit: {{ $d.gatewayMemoryLimit }}
  gatewayCPURequest: {{ $d.gatewayCPURequest }}
  gatewayCPULimit: {{ $d.gatewayCPULimit }}
  gatewayMemoryLimiterLimitMiB: {{ $gwLimiter.limitMiB }}
  gatewayMemoryLimiterSpikeLimitMiB: {{ $gwLimiter.spikeMiB }}
  gatewayGoMemLimitMiB: {{ $gwLimiter.goMiB }}

  nodeMemoryRequest: {{ $d.nodeMemoryRequest }}
  nodeMemoryLimit: {{ $d.nodeMemoryLimit }}
  nodeCPURequest: {{ $d.nodeCPURequest }}
  nodeCPULimit: {{ $d.nodeCPULimit }}
  nodeMemoryLimiterLimitMiB: {{ $nodeLimiter.limitMiB }}
  nodeMemoryLimiterSpikeLimitMiB: {{ $nodeLimiter.spikeMiB }}
  nodeGoMemLimitMiB: {{ $nodeLimiter.goMiB }}
  {{- end -}}


  {{/* ------------------------------------------------------------------ */
  /* 2) Gateway: effective CPU/Memory with mirroring rules                 */
  /* ------------------------------------------------------------------ */}}

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
  {{- $min := .Values.collectorGateway.minReplicas | default $d.gatewayMinReplicas -}}
  {{- $max := .Values.collectorGateway.maxReplicas | default $d.gatewayMaxReplicas -}}
  {{- if ge $min $max -}}
    {{- fail (printf "collectorGateway.minReplicas (%d) must be < collectorGateway.maxReplicas (%d)" $min $max) -}}
  {{- end -}}
  {{- $min -}}
  {{- end -}}

  {{- define "collector.gateway.maxReplicas" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- .Values.collectorGateway.maxReplicas | default $d.gatewayMaxReplicas -}}
  {{- end -}}


  {{/* ------------------------------------------------------------------ */
  /* 3) Node: effective CPU/Memory with mirroring rules                    */
  /* ------------------------------------------------------------------ */}}

  {{- define "collector.node.memoryRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "requestMemoryMiB" -}}
  {{- .Values.collectorNode.requestMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "limitMemoryMiB" -}}
  {{- .Values.collectorNode.limitMemoryMiB -}}
  {{- else -}}
  {{- $d.nodeMemoryRequest -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.memoryLimit" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "limitMemoryMiB" -}}
  {{- .Values.collectorNode.limitMemoryMiB -}}
  {{- else if hasKey .Values.collectorNode "requestMemoryMiB" -}}
  {{- .Values.collectorNode.requestMemoryMiB -}}
  {{- else -}}
  {{- $d.nodeMemoryLimit -}}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.cpuRequest" -}}
  {{- $d := include "collector.sizingDefaults" . | fromYaml -}}
  {{- if hasKey .Values.collectorNode "requestCPUm" -}}
  {{- .Values.collectorNode.requestCPUm -}}
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
  {{- else -}}
  {{- $d.nodeCPULimit -}}
  {{- end -}}
  {{- end -}}


  {{/* ------------------------------------------------------------------ */
  /* 4) Memory-limiter trios (gateway/node)                                */
  /*     Rule: if any of the trio is set → require all three               */
  /*     Else: derive all three from effective memory LIMIT                 */
  /* ------------------------------------------------------------------ */}}

  {{- define "collector.gateway.memoryLimiter" -}}
  {{- $v := .Values.collectorGateway | default dict -}}
  {{- $hasLimit := hasKey $v "memoryLimiterLimitMiB" -}}
  {{- $hasSpike := hasKey $v "memoryLimiterSpikeLimitMiB" -}}
  {{- $hasGo    := hasKey $v "goMemLimitMiB" -}}
  {{- if or $hasLimit $hasSpike $hasGo -}}
    {{- if not (and $hasLimit $hasSpike $hasGo) -}}
      {{- fail "collectorGateway: if any of memoryLimiterLimitMiB/memoryLimiterSpikeLimitMiB/goMemLimitMiB is set, all three must be set" -}}
    {{- end -}}
  limitMiB: {{ $v.memoryLimiterLimitMiB }}
  spikeMiB: {{ $v.memoryLimiterSpikeLimitMiB }}
  goMiB: {{ $v.goMemLimitMiB }}
  {{- else -}}
    {{- $memLimit := include "collector.gateway.memoryLimit" . | int -}}
    {{- $hard := sub $memLimit 50 -}}
    {{- $spike := div (mul $hard 20) 100 -}}
    {{- $go := div (mul $hard 80) 100 -}}
  limitMiB: {{ $hard }}
  spikeMiB: {{ $spike }}
  goMiB: {{ $go }}
  {{- end -}}
  {{- end -}}

  {{- define "collector.node.memoryLimiter" -}}
  {{- $v := .Values.collectorNode | default dict -}}
  {{- $hasLimit := hasKey $v "memoryLimiterLimitMiB" -}}
  {{- $hasSpike := hasKey $v "memoryLimiterSpikeLimitMiB" -}}
  {{- $hasGo    := hasKey $v "goMemLimitMiB" -}}
  {{- if or $hasLimit $hasSpike $hasGo -}}
    {{- if not (and $hasLimit $hasSpike $hasGo) -}}
      {{- fail "collectorNode: if any of memoryLimiterLimitMiB/memoryLimiterSpikeLimitMiB/goMemLimitMiB is set, all three must be set" -}}
    {{- end -}}
  limitMiB: {{ $v.memoryLimiterLimitMiB }}
  spikeMiB: {{ $v.memoryLimiterSpikeLimitMiB }}
  goMiB: {{ $v.goMemLimitMiB }}
  {{- else -}}
    {{- $memLimit := include "collector.node.memoryLimit" . | int -}}
    {{- $hard := sub $memLimit 50 -}}
    {{- $spike := div (mul $hard 20) 100 -}}
    {{- $go := div (mul $hard 80) 100 -}}
  limitMiB: {{ $hard }}
  spikeMiB: {{ $spike }}
  goMiB: {{ $go }}
  {{- end -}}
  {{- end -}}
