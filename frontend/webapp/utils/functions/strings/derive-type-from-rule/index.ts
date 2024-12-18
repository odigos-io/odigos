import { type InstrumentationRuleInput, InstrumentationRuleType, type InstrumentationRuleSpec } from '@/types';

export const deriveTypeFromRule = (rule: InstrumentationRuleInput | InstrumentationRuleSpec) => {
  if (rule.payloadCollection) {
    return InstrumentationRuleType.PAYLOAD_COLLECTION;
  }

  return InstrumentationRuleType.UNKNOWN_TYPE;
};
