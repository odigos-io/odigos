import { InstrumentationRuleType, type InstrumentationRuleSpec } from '@/types';

const deriveTypeFromRule = (rule: InstrumentationRuleSpec): InstrumentationRuleType | undefined => {
  if (rule.payloadCollection) {
    return InstrumentationRuleType.PAYLOAD_COLLECTION;
  }

  return undefined;
};

export { deriveTypeFromRule };
