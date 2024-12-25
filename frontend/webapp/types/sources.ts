import { type Condition } from './common';
import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export type SourceContainer = {
  containerName: string;
  language: WORKLOAD_PROGRAMMING_LANGUAGES;
  runtimeVersion: string;
  otherAgent: string | null;
};

export type K8sActualSource = {
  name: string;
  kind: string;
  namespace: string;
  reportedName: string;
  numberOfInstances: number;
  selected?: boolean;
  instrumentedApplicationDetails: {
    containers: Array<SourceContainer>;
    conditions: Array<Condition>;
  };
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

export interface Source {
  spec: {
    workload: WorkloadId;
  };
  status: {
    conditions: Condition[];
  };
}
