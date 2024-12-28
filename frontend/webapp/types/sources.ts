import { type Condition } from './common';
import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export type SourceContainer = {
  containerName: string;
  language: WORKLOAD_PROGRAMMING_LANGUAGES;
  runtimeVersion: string;
  otherAgent: string | null;
};

export type K8sActualSource = {
  namespace: string;
  name: string;
  kind: string;
  numberOfInstances: number;
  // serviceName: string;
  reportedName: string;
  containers: Array<SourceContainer>;
  conditions: Array<Condition>;
  selected?: boolean; // not from backend
};

export type WorkloadId = {
  kind: string;
  name: string;
  namespace: string;
};

export interface PatchSourceRequestInput {
  reportedName?: string;
}

export type PersistSourcesArray = {
  kind: string;
  name: string;
  selected: boolean;
};
