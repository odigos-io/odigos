import { type InstrumentationRuleInput, InstrumentationRuleType, type InstrumentationRuleSpec } from '@/types';

export const deriveTypeFromRule = (rule: InstrumentationRuleInput | InstrumentationRuleSpec): InstrumentationRuleType | undefined => {
  if (rule.payloadCollection) {
    return InstrumentationRuleType.PAYLOAD_COLLECTION;
  }

  return undefined;
};
