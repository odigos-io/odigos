import type { ActionData } from './actions';
import type { K8sActualSource } from './sources';
import type { ActualDestination } from './destinations';
import type { InstrumentationRuleSpec } from './instrumentation-rules';

export interface TokenPayload {
  token: string;
  name: string;
  issuedAt: number;
  expiresAt: number;
}

export interface K8sActualNamespace {
  name: string;
  selected: boolean;
  k8sActualSources?: K8sActualSource[];
}

interface PaginatedSources {
  nextPage: string;
  items: K8sActualSource[];
}

interface ComputePlatformData {
  computePlatformType?: string;
  apiTokens?: TokenPayload[];
  k8sActualNamespaces?: K8sActualNamespace[];
  k8sActualNamespace?: K8sActualNamespace;
  sources?: PaginatedSources;
  destinations?: ActualDestination[];
  actions?: ActionData[]; // mapped to "ActionDataParsed" in the frontend
  instrumentationRules?: InstrumentationRuleSpec[]; // mapped to "InstrumentationRuleSpecMapped" in the frontend
}

export type ComputePlatform = {
  computePlatform: ComputePlatformData;
};
