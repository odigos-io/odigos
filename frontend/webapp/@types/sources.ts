import type { Source } from '@odigos/ui-utils';

export interface FetchedSource extends Source {}

export interface SourceInstrumentInput {
  namespace: string;
  sources: Pick<Source, 'name' | 'kind' | 'selected'>[];
}

export interface SourceUpdateInput {
  otelServiceName: string;
}
