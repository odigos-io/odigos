import type { InstrumentationRule } from '@odigos/ui-kit/types';

export type FetchedInstrumentationRule = Omit<InstrumentationRule, 'type'>;
