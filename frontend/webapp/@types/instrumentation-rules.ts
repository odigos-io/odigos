import type { InstrumentationRule, WorkloadId } from '@odigos/ui-utils';

export interface FetchedInstrumentationRule {
  ruleId: string;
  ruleName: string;
  notes: string;
  disabled: boolean;
  mutable: boolean;
  profileName: string;
  workloads?: WorkloadId[] | null;
  instrumentationLibraries?: { language: string; library: string }[] | null;
  payloadCollection: InstrumentationRule['payloadCollection']; // TODO: define this locally
  codeAttributes: InstrumentationRule['codeAttributes']; // TODO: define this locally
}
