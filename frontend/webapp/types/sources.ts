import { type Source } from '@odigos/ui-containers';
import { type FetchedCondition } from '@odigos/ui-utils';

export interface FetchedSource extends Source {
  conditions: FetchedCondition[] | null;
}

export interface FetchedAvailableSources {
  [namespace: string]: Pick<FetchedSource, 'name' | 'kind' | 'selected' | 'numberOfInstances'>[];
}

export interface SourceInstrumentInput {
  [namespace: string]: Pick<FetchedSource, 'name' | 'kind' | 'selected'>[];
}

export interface SourceUpdateInput {
  otelServiceName: string;
}
