import type { SelectedSource } from '@odigos/ui-kit/store';
import type { Condition, WorkloadId } from '@odigos/ui-kit/types';

export interface SourceInstrumentInput {
  sources: Omit<SelectedSource, 'numberOfInstances'>[];
}

export type SourceConditions = WorkloadId & { conditions: Condition[] };
