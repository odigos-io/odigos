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
  otelServerName: string;
  containers: Array<SourceContainer>;
  conditions: Array<Condition>;
}

export interface PatchSourceRequestInput {
  otelServerName?: string;
}

export interface PersistSourcesArray {
  kind: string;
  name: string;
  selected: boolean;
}
