import type { Source, FetchedCondition } from '@odigos/ui-utils';

export interface FetchedSource extends Source {
  conditions: FetchedCondition[] | null;
}

export interface SourceInstrumentInput {
  namespace: string;
  sources: Pick<Source, 'name' | 'kind' | 'selected'>[];
}

export interface SourceUpdateInput {
  otelServiceName: string;
}
