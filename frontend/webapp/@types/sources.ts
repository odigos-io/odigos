import type { Source } from '@odigos/ui-kit/types';

export interface FetchedSource extends Source {}

export interface SourceInstrumentInput {
  namespace: string;
  sources: Pick<Source, 'name' | 'kind' | 'selected'>[];
}

export interface SourceUpdateInput {
  otelServiceName: string;
}
