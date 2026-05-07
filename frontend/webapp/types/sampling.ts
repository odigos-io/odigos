export interface SourcesScope {
  workloadName?: string | null;
  workloadKind?: string | null;
  workloadNamespace?: string | null;
  workloadLanguage?: string | null;
}

export interface SourcesScopeInput {
  workloadName?: string | null;
  workloadKind?: string | null;
  workloadNamespace?: string | null;
  workloadLanguage?: string | null;
}

export interface HeadSamplingHttpServerMatcher {
  route?: string | null;
  routePrefix?: string | null;
  method?: string | null;
}

export interface HeadSamplingHttpClientMatcher {
  serverAddress?: string | null;
  templatedPath?: string | null;
  templatedPathPrefix?: string | null;
  method?: string | null;
}

export interface HeadSamplingOperationMatcher {
  httpServer?: HeadSamplingHttpServerMatcher | null;
  httpClient?: HeadSamplingHttpClientMatcher | null;
}

export interface TailSamplingHttpServerMatcher {
  route?: string | null;
  routePrefix?: string | null;
  method?: string | null;
}

export interface TailSamplingKafkaMatcher {
  kafkaTopic?: string | null;
}

export interface TailSamplingOperationMatcher {
  httpServer?: TailSamplingHttpServerMatcher | null;
  kafkaConsumer?: TailSamplingKafkaMatcher | null;
  kafkaProducer?: TailSamplingKafkaMatcher | null;
}

export interface NoisyOperationRule {
  ruleId: string;
  name?: string | null;
  disabled: boolean;
  sourceScopes?: SourcesScope[] | null;
  operation?: HeadSamplingOperationMatcher | null;
  percentageAtMost?: number | null;
  notes?: string | null;
}

export interface HighlyRelevantOperationRule {
  ruleId: string;
  name?: string | null;
  disabled: boolean;
  sourceScopes?: SourcesScope[] | null;
  error: boolean;
  durationAtLeastMs?: number | null;
  operation?: TailSamplingOperationMatcher | null;
  percentageAtLeast?: number | null;
  notes?: string | null;
}

export interface CostReductionRule {
  ruleId: string;
  name?: string | null;
  disabled: boolean;
  sourceScopes?: SourcesScope[] | null;
  operation?: TailSamplingOperationMatcher | null;
  percentageAtMost: number;
  notes?: string | null;
}

export interface SamplingRules {
  id: string;
  name?: string | null;
  noisyOperations: NoisyOperationRule[];
  highlyRelevantOperations: HighlyRelevantOperationRule[];
  costReductionRules: CostReductionRule[];
}

export type SamplingRuleType = 'noisy' | 'highlyRelevant' | 'costReduction';

export type SamplingRule = NoisyOperationRule | HighlyRelevantOperationRule | CostReductionRule;

export interface NoisyOperationRuleInput {
  name?: string | null;
  disabled?: boolean | null;
  sourceScopes?: SourcesScopeInput[] | null;
  operation?: HeadSamplingOperationMatcher | null;
  percentageAtMost?: number | null;
  notes?: string | null;
}

export interface HighlyRelevantOperationRuleInput {
  name?: string | null;
  disabled?: boolean | null;
  sourceScopes?: SourcesScopeInput[] | null;
  error?: boolean | null;
  durationAtLeastMs?: number | null;
  operation?: TailSamplingOperationMatcher | null;
  percentageAtLeast?: number | null;
  notes?: string | null;
}

export interface CostReductionRuleInput {
  name?: string | null;
  disabled?: boolean | null;
  sourceScopes?: SourcesScopeInput[] | null;
  operation?: TailSamplingOperationMatcher | null;
  percentageAtMost: number;
  notes?: string | null;
}

export interface K8sHealthProbesSamplingConfig {
  enabled: boolean | null;
  keepPercentage: number | null;
}

export interface SamplingQueryResponse {
  sampling: {
    configs: {
      effective: {
        k8sHealthProbesSampling: K8sHealthProbesSamplingConfig | null;
      };
    };
    rules: SamplingRules[];
  };
}
