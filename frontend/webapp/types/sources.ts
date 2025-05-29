import type { SelectedSource } from '@odigos/ui-kit/store';
import type { Condition, WorkloadId } from '@odigos/ui-kit/types';

export interface SourceInstrumentInput {
  namespace: string;
  sources: Omit<SelectedSource, 'numberOfInstances'>[];
}

export type InstrumentationInstancesHealth = WorkloadId & {
  condition: Condition;
};
