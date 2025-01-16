import { type Condition } from './common';
import { WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';

export enum K8sResourceKind {
  Deployment = 'Deployment',
  DaemonSet = 'DaemonSet',
  StatefulSet = 'StatefulSet',
}

export type WorkloadId = {
  namespace: string;
  name: string;
  kind: K8sResourceKind;
};

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
  selected: boolean;
  reportedName: string;
  containers: Array<SourceContainer>;
  conditions: Array<Condition>;
};

export interface PatchSourceRequestInput {
  reportedName?: string;
}

export type PersistSourcesArray = {
  kind: string;
  name: string;
  selected: boolean;
};
