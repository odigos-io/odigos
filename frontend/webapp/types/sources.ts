import type { SelectedSource } from '@odigos/ui-kit/store';

export interface SourceInstrumentInput {
  sources: Omit<SelectedSource, 'numberOfInstances'>[];
}
