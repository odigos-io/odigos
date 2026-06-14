'use client';

/**
 * `<OdigosApiAdapter>` — wires the kit's `<OdigosApiProvider>` to the
 * webapp's GraphQL backend.
 *
 * Owns:
 *   1. CSRF token bootstrap (renders a `<FadeLoader />` until ready, then
 *      injects the token via `apolloConfig.csrfHeader`).
 *   2. GraphQL operation map — every kit operation maps to a `gql` document
 *      from `@/graphql`. No `transformVariables` / `transformResult` needed
 *      because the standalone backend already returns the kit's expected shape.
 *   3. Operation context derived from `useOdigos()` (platform, tier, version).
 *
 * Mounts at the layout level once. Pages stay one-liners:
 *   `export default function Page() { return <Overview metrics={…} />; }`
 */

import React, { type FC, type PropsWithChildren, useMemo } from 'react';
import { useCSRF } from '@/hooks';
import { API, IS_LOCAL } from '@/utils';
import { PlatformType, Tier } from '@odigos/ui-kit/types';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';
import {
  OdigosApiProvider,
  type DiagnoseResult,
  type GetNamespacesWithWorkloadsData,
  type GetSamplingRulesData,
  type GetServiceMapData,
  type OdigosApiOperations,
  type OperationContext,
} from '@odigos/ui-kit/contexts/odigos-api';
import type {
  EffectiveConfig,
  EnableProfilingResult,
  ExtendedPodInfo,
  FetchedConfig,
  GatewayInfo,
  Namespace,
  NodeCollectoInfo,
  PodInfo,
  ProfilingSlots,
  SamplingRules,
  SamplingRulesK8sHealthConfig,
  ServiceMapSources,
  SourceProfilingResult,
  TestConnectionResponse,
  TokenPayload,
} from '@odigos/ui-kit/types';
import {
  // queries
  GET_ACTIONS,
  GET_CONFIG,
  GET_CONFIG_YAMLS,
  GET_DATA_STREAMS,
  GET_DESTINATIONS,
  GET_DESTINATION_CATEGORIES,
  GET_EFFECTIVE_CONFIG,
  GET_INSTRUMENTATION_RULES,
  GET_K8S_MANIFEST,
  GET_METRICS,
  GET_NAMESPACES_WITH_WORKLOADS,
  GET_PEER_SOURCES,
  GET_POTENTIAL_DESTINATIONS,
  GET_PROFILING_SLOTS,
  GET_SAMPLING_RULES,
  GET_SERVICE_MAP,
  GET_SOURCE,
  GET_SOURCE_LIBRARIES,
  GET_SOURCE_PROFILING,
  GET_TOKENS,
  GET_WORKLOADS,
  GET_WORKLOADS_BY_IDS,
  GET_WORKLOADS_BY_IDS_SLIM,
  GET_GATEWAY_INFO,
  GET_GATEWAY_PODS,
  GET_NODE_COLLECTOR_INFO,
  GET_NODE_COLLECTOR_PODS,
  GET_COLLECTOR_POD_INFO,
  DESCRIBE_ODIGOS,
  DESCRIBE_SOURCE,
  DOWNLOAD_DIAGNOSE,
  // mutations
  CREATE_ACTION,
  CREATE_COST_REDUCTION_RULE,
  CREATE_DESTINATION,
  CREATE_HIGHLY_RELEVANT_OPERATION_RULE,
  CREATE_INSTRUMENTATION_RULE,
  CREATE_NOISY_OPERATION_RULE,
  DELETE_ACTION,
  DELETE_COST_REDUCTION_RULE,
  DELETE_DATA_STREAM,
  DELETE_DESTINATION,
  DELETE_HIGHLY_RELEVANT_OPERATION_RULE,
  DELETE_INSTRUMENTATION_RULE,
  DELETE_NOISY_OPERATION_RULE,
  ENABLE_SOURCE_PROFILING,
  PERSIST_NAMESPACES,
  PERSIST_SOURCES,
  RECOVER_FROM_ROLLBACK,
  RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS,
  RESTART_POD,
  RESTART_WORKLOADS,
  TEST_CONNECTION_MUTATION,
  UPDATE_ACTION,
  UPDATE_API_TOKEN,
  UPDATE_COST_REDUCTION_RULE,
  UPDATE_DATA_STREAM,
  UPDATE_DESTINATION,
  UPDATE_HIGHLY_RELEVANT_OPERATION_RULE,
  UPDATE_INSTRUMENTATION_RULE,
  UPDATE_K8S_ACTUAL_SOURCE,
  UPDATE_LOCAL_UI_CONFIG,
  UPDATE_LOCAL_UI_SAMPLING_CONFIG,
  UPDATE_NOISY_OPERATION_RULE,
} from '@/graphql';

// The CREATE_DATA_STREAM op is intentionally absent: the standalone backend
// doesn't expose a dedicated mutation; the kit's `dataStreams.create` falls
// back to local-store-only behavior when the operation slot is undefined.

// Stable operations map — referentially constant so the kit's runner doesn't
// rebuild memoized callbacks on every render.
const operations: OdigosApiOperations = {
  // sources / workloads
  GET_WORKLOADS: { document: GET_WORKLOADS },
  GET_WORKLOADS_BY_IDS: { document: GET_WORKLOADS_BY_IDS },
  GET_WORKLOADS_BY_IDS_SLIM: { document: GET_WORKLOADS_BY_IDS_SLIM },
  GET_SOURCE: { document: GET_SOURCE },
  GET_SOURCE_LIBRARIES: { document: GET_SOURCE_LIBRARIES },
  GET_PEER_SOURCES: { document: GET_PEER_SOURCES },
  PERSIST_SOURCES: { document: PERSIST_SOURCES },
  UPDATE_SOURCE: { document: UPDATE_K8S_ACTUAL_SOURCE },
  RESTART_WORKLOADS: { document: RESTART_WORKLOADS },
  RESTART_POD: { document: RESTART_POD },
  RECOVER_FROM_ROLLBACK: { document: RECOVER_FROM_ROLLBACK },

  // destinations
  GET_DESTINATIONS: { document: GET_DESTINATIONS },
  GET_DESTINATION_CATEGORIES: { document: GET_DESTINATION_CATEGORIES },
  GET_POTENTIAL_DESTINATIONS: { document: GET_POTENTIAL_DESTINATIONS },
  CREATE_DESTINATION: { document: CREATE_DESTINATION },
  UPDATE_DESTINATION: { document: UPDATE_DESTINATION },
  DELETE_DESTINATION: { document: DELETE_DESTINATION },
  TEST_DESTINATION_CONNECTION: {
    document: TEST_CONNECTION_MUTATION,
    // The webapp's mutation returns `{ testConnectionForDestination: TestConnectionResponse }`;
    // unwrap so consumers get the bare response shape that the kit hook returns.
    transformResult: (raw) => (raw as { testConnectionForDestination?: TestConnectionResponse } | null | undefined)?.testConnectionForDestination,
  },

  // actions
  GET_ACTIONS: { document: GET_ACTIONS },
  CREATE_ACTION: { document: CREATE_ACTION },
  UPDATE_ACTION: { document: UPDATE_ACTION },
  DELETE_ACTION: { document: DELETE_ACTION },

  // instrumentation rules
  GET_INSTRUMENTATION_RULES: { document: GET_INSTRUMENTATION_RULES },
  CREATE_INSTRUMENTATION_RULE: { document: CREATE_INSTRUMENTATION_RULE },
  UPDATE_INSTRUMENTATION_RULE: { document: UPDATE_INSTRUMENTATION_RULE },
  DELETE_INSTRUMENTATION_RULE: { document: DELETE_INSTRUMENTATION_RULE },

  // data streams
  GET_DATA_STREAMS: { document: GET_DATA_STREAMS },
  UPDATE_DATA_STREAM: { document: UPDATE_DATA_STREAM },
  DELETE_DATA_STREAM: { document: DELETE_DATA_STREAM },

  // namespaces
  // Wire returns `{ namespaces: [...] }` flat at the root (the legacy
  // `computePlatform.k8sActualNamespaces` path was retired in v1.20).
  // The kit's slot already expects `{ namespaces }` directly, so the
  // raw wire shape passes through — but we still declare a no-op
  // `transformResult` for symmetry with central-ui's adapter, and to
  // keep the consumer's TS narrowing happy (slot data is bare).
  GET_NAMESPACES_WITH_WORKLOADS: {
    document: GET_NAMESPACES_WITH_WORKLOADS,
    transformResult: (raw): GetNamespacesWithWorkloadsData => ({
      namespaces: (raw as { namespaces?: Namespace[] } | null | undefined)?.namespaces ?? [],
    }),
  },
  PERSIST_NAMESPACES: { document: PERSIST_NAMESPACES },

  // k8s manifest. Bare slot expects a single string. Wire field is
  // `k8sManifest`; we keep the legacy `k8sActualManifest`/`manifest`
  // fallbacks too in case an older backend ships either of those names.
  GET_K8S_MANIFEST: {
    document: GET_K8S_MANIFEST,
    transformVariables: (vars) => ({ kind: vars?.kind, name: vars?.name, namespace: vars?.namespace, ext: vars?.ext ?? 'yaml' }),
    transformResult: (raw: unknown) => {
      if (typeof raw === 'string') return raw;
      const obj = raw as { k8sManifest?: string; k8sActualManifest?: string; manifest?: string } | null | undefined;
      return obj?.k8sManifest ?? obj?.k8sActualManifest ?? obj?.manifest;
    },
  },

  // config
  GET_CONFIG: {
    document: GET_CONFIG,
    // Wire returns `{ config: FetchedConfig }`; the kit's slot exposes
    // the bare `FetchedConfig`, so flatten here.
    transformResult: (raw: unknown) => (raw as { config?: FetchedConfig } | null | undefined)?.config,
  },
  GET_EFFECTIVE_CONFIG: {
    document: GET_EFFECTIVE_CONFIG,
    transformResult: (raw) => {
      const env = raw as { computePlatform?: { effectiveConfig?: EffectiveConfig }; effectiveConfig?: EffectiveConfig } | null | undefined;
      return env?.computePlatform?.effectiveConfig ?? env?.effectiveConfig;
    },
  },
  GET_CONFIG_YAMLS: { document: GET_CONFIG_YAMLS },
  UPDATE_LOCAL_UI_CONFIG: { document: UPDATE_LOCAL_UI_CONFIG },
  RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS: { document: RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS },

  // describe / diagnose. Bare-shape slot for diagnose unwraps the
  // wire envelope here so consumers get a typed `DiagnoseResult` directly.
  GET_DESCRIBE_ODIGOS: { document: DESCRIBE_ODIGOS },
  GET_DESCRIBE_SOURCE: { document: DESCRIBE_SOURCE },
  GET_DIAGNOSE: {
    document: DOWNLOAD_DIAGNOSE,
    transformResult: (raw: unknown) => (raw as { diagnose?: DiagnoseResult } | null | undefined)?.diagnose,
  },

  // tokens. Bare `TokenPayload[]` slot — flatten the wire envelope.
  // Wire returns `{ computePlatform: { apiTokens: [...] } }`; we also
  // accept the legacy `tokens` field name as a fallback in case an
  // older backend version is in play.
  GET_TOKENS: {
    document: GET_TOKENS,
    transformResult: (raw: unknown) => {
      const env = raw as
        | {
            apiTokens?: TokenPayload[];
            tokens?: TokenPayload[];
            computePlatform?: { apiTokens?: TokenPayload[]; tokens?: TokenPayload[] };
          }
        | null
        | undefined;
      return env?.computePlatform?.apiTokens ?? env?.computePlatform?.tokens ?? env?.apiTokens ?? env?.tokens ?? [];
    },
  },
  UPDATE_TOKEN: { document: UPDATE_API_TOKEN },

  // metrics / service map
  GET_METRICS: { document: GET_METRICS },
  // Wire returns `{ getServiceMap: { services: [...] } }`; the kit's
  // `ServiceMapApi.fetch()` consumer reads `data.serviceMap`. Flatten
  // to that shape so the consumer stays envelope-agnostic.
  GET_SERVICE_MAP: {
    document: GET_SERVICE_MAP,
    transformResult: (raw): GetServiceMapData => {
      const env = raw as { getServiceMap?: { services?: ServiceMapSources }; serviceMap?: ServiceMapSources } | null | undefined;
      return { serviceMap: env?.getServiceMap?.services ?? env?.serviceMap ?? [] };
    },
  },

  // profiling — bare-shape slots; flatten the per-field envelope here.
  GET_PROFILING_SLOTS: {
    document: GET_PROFILING_SLOTS,
    transformResult: (raw: unknown) => (raw as { profilingSlots?: ProfilingSlots } | null | undefined)?.profilingSlots,
  },
  GET_SOURCE_PROFILING: {
    document: GET_SOURCE_PROFILING,
    // Wire returns `{ computePlatform: { source: { profiling: { profileJson } } } }`.
    // Walk the path and surface the bare `{ profileJson }` shape so the
    // consumer never has to narrow.
    transformResult: (raw: unknown) => {
      const env = raw as { computePlatform?: { source?: { profiling?: SourceProfilingResult } }; sourceProfiling?: SourceProfilingResult } | null | undefined;
      return env?.computePlatform?.source?.profiling ?? env?.sourceProfiling;
    },
  },
  ENABLE_SOURCE_PROFILING: {
    document: ENABLE_SOURCE_PROFILING,
    transformResult: (raw: unknown) => (raw as { enableSourceProfiling?: EnableProfilingResult } | null | undefined)?.enableSourceProfiling,
  },

  // pipeline collectors — bare-shape slots. The standalone backend's
  // GraphQL field names predate the kit's normalization: the gateway
  // is exposed as `gatewayDeploymentInfo` (it's a Deployment), the
  // node-collector is exposed as `odigletDaemonSetInfo` / `odigletPods`
  // (it's an Odiglet DaemonSet). The kit's slot type is bare
  // `GatewayInfo` / `NodeCollectoInfo` / `PodInfo[]`, so extract from
  // the wire field names here so consumers see the canonical shape.
  GET_GATEWAY_INFO: {
    document: GET_GATEWAY_INFO,
    transformResult: (raw: unknown) => (raw as { gatewayDeploymentInfo?: GatewayInfo } | null | undefined)?.gatewayDeploymentInfo,
  },
  GET_GATEWAY_PODS: {
    document: GET_GATEWAY_PODS,
    transformResult: (raw: unknown) => (raw as { gatewayPods?: PodInfo[] } | null | undefined)?.gatewayPods ?? [],
  },
  GET_NODE_COLLECTOR_INFO: {
    document: GET_NODE_COLLECTOR_INFO,
    transformResult: (raw: unknown) => (raw as { odigletDaemonSetInfo?: NodeCollectoInfo } | null | undefined)?.odigletDaemonSetInfo,
  },
  GET_NODE_COLLECTOR_PODS: {
    document: GET_NODE_COLLECTOR_PODS,
    transformResult: (raw: unknown) => (raw as { odigletPods?: PodInfo[] } | null | undefined)?.odigletPods ?? [],
  },
  GET_COLLECTOR_POD_INFO: {
    document: GET_COLLECTOR_POD_INFO,
    // Wire field is `collectorPod`. The legacy `extendedPodInfo`
    // fallback is kept for any older backend version that might still
    // be in flight.
    transformResult: (raw: unknown) =>
      (raw as { collectorPod?: ExtendedPodInfo; extendedPodInfo?: ExtendedPodInfo } | null | undefined)?.collectorPod ??
      (raw as { extendedPodInfo?: ExtendedPodInfo } | null | undefined)?.extendedPodInfo,
  },

  // sampling
  // Wire returns `{ sampling: { rules: [...], configs: { effective: {
  // k8sHealthProbesSampling: { enabled, keepPercentage } } } } }`. The
  // kit's slot expects the bare `{ samplingRules, k8sHealthProbesConfig }`
  // shape; flatten here so consumers don't have to dig through nested
  // envelopes.
  GET_SAMPLING_RULES: {
    document: GET_SAMPLING_RULES,
    transformResult: (raw): GetSamplingRulesData => {
      const env = raw as
        | {
            sampling?: {
              rules?: SamplingRules[];
              configs?: { effective?: { k8sHealthProbesSampling?: SamplingRulesK8sHealthConfig } };
            };
          }
        | null
        | undefined;
      return {
        samplingRules: env?.sampling?.rules ?? [],
        k8sHealthProbesConfig: env?.sampling?.configs?.effective?.k8sHealthProbesSampling ?? undefined,
      };
    },
  },
  CREATE_NOISY_OPERATION_RULE: { document: CREATE_NOISY_OPERATION_RULE },
  UPDATE_NOISY_OPERATION_RULE: { document: UPDATE_NOISY_OPERATION_RULE },
  DELETE_NOISY_OPERATION_RULE: { document: DELETE_NOISY_OPERATION_RULE },
  CREATE_HIGHLY_RELEVANT_OPERATION_RULE: { document: CREATE_HIGHLY_RELEVANT_OPERATION_RULE },
  UPDATE_HIGHLY_RELEVANT_OPERATION_RULE: { document: UPDATE_HIGHLY_RELEVANT_OPERATION_RULE },
  DELETE_HIGHLY_RELEVANT_OPERATION_RULE: { document: DELETE_HIGHLY_RELEVANT_OPERATION_RULE },
  CREATE_COST_REDUCTION_RULE: { document: CREATE_COST_REDUCTION_RULE },
  UPDATE_COST_REDUCTION_RULE: { document: UPDATE_COST_REDUCTION_RULE },
  DELETE_COST_REDUCTION_RULE: { document: DELETE_COST_REDUCTION_RULE },
  UPDATE_LOCAL_UI_SAMPLING_CONFIG: { document: UPDATE_LOCAL_UI_SAMPLING_CONFIG },
};

/**
 * Initial operation context used while we bootstrap. The real context
 * (platformType/tier/version) is derived by the inner layout via
 * `useConfig` and propagated through React rerenders.
 */
const INITIAL_CONTEXT: OperationContext = {
  platformType: PlatformType.K8s,
  tier: Tier.Community,
  version: 'v0.0.0',
};

interface AdapterProps extends PropsWithChildren {
  /**
   * Operation context override. Inner layout supplies the real values
   * after fetching `GET_CONFIG`. Memoize at the call site to avoid
   * re-creating the kit's internal context value on each render.
   */
  context?: OperationContext;
}

const OdigosApiAdapter: FC<AdapterProps> = ({ children, context }) => {
  const { token, isLoading } = useCSRF();

  // In local dev, CSRF is disabled — proceed without a token.
  // In production we wait for the bootstrap fetch to complete before
  // mounting the kit's provider so all subsequent requests include the
  // header. This mirrors the previous `apollo-provider.tsx` behavior.
  const ready = IS_LOCAL || !isLoading;

  const resolvedContext = context ?? INITIAL_CONTEXT;

  const apolloConfig = useMemo(
    () => ({
      httpUrl: API.GRAPHQL,
      credentials: (IS_LOCAL ? 'same-origin' : 'include') as 'same-origin' | 'include',
      // Synchronously read the cached token (resolved before this provider mounts).
      csrfHeader: (): Record<string, string> => (token ? { 'X-CSRF-Token': token } : {}),
      addTypename: false,
      defaultFetchPolicies: {
        watchQuery: 'cache-and-network' as const,
        query: 'cache-first' as const,
        mutate: 'network-only' as const,
      },
    }),
    [token],
  );

  if (!ready) {
    return (
      <CenterThis style={{ height: '100%' }}>
        <FadeLoader scale={2} />
      </CenterThis>
    );
  }

  return (
    <OdigosApiProvider apolloConfig={apolloConfig} operations={operations} context={resolvedContext}>
      {children}
    </OdigosApiProvider>
  );
};

export default OdigosApiAdapter;
