import { FetchedSource } from './sources';

export interface FetchedNamespace {
  name: string;
  selected: boolean;
  k8sActualSources?: FetchedSource[];
}

export interface NamespaceInstrumentInput {
  name: string;
  futureSelected: boolean;
}
