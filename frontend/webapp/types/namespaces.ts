import type { Source } from '@odigos/ui-kit/types';

export interface FetchedNamespace {
  name: string;
  selected: boolean;
  sources?: Source[];
}

export interface NamespaceInstrumentInput {
  name: string;
  futureSelected: boolean;
}
