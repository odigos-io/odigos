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
<<<<<<< HEAD
};

export interface PatchSourceRequestInput {
  reportedName?: string;
}
=======
};

export interface PatchSourceRequestInput {
  reportedName?: string;
}

export type PersistSourcesArray = {
  kind: string;
  name: string;
  selected: boolean;
};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
