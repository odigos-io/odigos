import { type Condition } from './common';
import { PROGRAMMING_LANGUAGES, WorkloadId } from '@odigos/ui-components';

export interface SourceContainer {
  containerName: string;
  language: PROGRAMMING_LANGUAGES;
  runtimeVersion: string;
  otherAgent: string | null;
}

export interface K8sActualSource extends WorkloadId {
  selected: boolean;
  numberOfInstances?: number;
  otelServiceName: string;
  containers: Array<SourceContainer>;
  conditions: Array<Condition>;
}

export interface PatchSourceRequestInput {
  otelServiceName: string;
}

export interface PersistSourcesArray {
  kind: string;
  name: string;
  selected: boolean;
}
