import type { Source, Condition, WorkloadId } from '@odigos/ui-kit/types';

export interface SourceInstrumentInput {
  namespace: string;
  sources: Pick<Source, 'name' | 'kind' | 'selected'>[];
}

export interface SourceUpdateInput {
  otelServiceName: string;
}

export type InstrumentationInstancesHealth = WorkloadId & {
  condition: Condition;
};
