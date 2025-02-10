import type { Source, FetchedCondition } from '@odigos/ui-utils';

export interface FetchedSource extends Source {
  conditions: FetchedCondition[] | null;
}

export interface SourceUpdateInput {
  otelServiceName: string;
}
