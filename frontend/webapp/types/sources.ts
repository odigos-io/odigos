import { Condition } from './common';

export type SourceContainer = {
  containerName: string;
  language: string;
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
