import { InstrumentationRuleType, type InstrumentationRuleSpec } from '@/types';

export const deriveTypeFromRule = (rule: InstrumentationRuleSpec): InstrumentationRuleType | undefined => {
  if (rule.payloadCollection) {
    return InstrumentationRuleType.PAYLOAD_COLLECTION;
  }

  return undefined;
};

// TODO: add "deriveTypeFromAction"
