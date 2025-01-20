import { type Condition } from './common';
import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export enum K8sResourceKind {
  Deployment = 'Deployment',
  DaemonSet = 'DaemonSet',
  StatefulSet = 'StatefulSet',
}

export interface WorkloadId {
  namespace: string;
  name: string;
  kind: string; // TODO: replace with "K8sResourceKind" and fix all TS errors;
}

export interface SourceContainer {
  containerName: string;
  language: WORKLOAD_PROGRAMMING_LANGUAGES;
  runtimeVersion: string;
  otherAgent: string | null;
}

export interface K8sActualSource extends WorkloadId {
  numberOfInstances: number;
  selected: boolean;
  reportedName: string;
  containers: Array<SourceContainer>;
  conditions: Array<Condition>;
}

export interface PatchSourceRequestInput {
  sourceId: WorkloadId;
  patchSourceRequest: {
    reportedName?: string;
  };
}

export interface PersistSourcesInput {
  namespace: string;
  sources: {
    name: WorkloadId['name'];
    kind: WorkloadId['kind'];
    selected: boolean;
  }[];
}
