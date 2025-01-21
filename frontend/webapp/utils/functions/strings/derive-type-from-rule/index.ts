import { type InstrumentationRuleInput, InstrumentationRuleType, type InstrumentationRuleSpec } from '@/types';

export const deriveTypeFromRule = (rule: InstrumentationRuleInput | InstrumentationRuleSpec) => {
  const allKeysAreNull = (obj: Record<string, any>) => Object.values(obj).every((v) => v === null);

  if (rule.payloadCollection && !allKeysAreNull(rule.payloadCollection)) {
    return InstrumentationRuleType.PAYLOAD_COLLECTION;
  } else if (rule.codeAttributes && !allKeysAreNull(rule.codeAttributes)) {
    return InstrumentationRuleType.CODE_ATTRIBUTES;
  }

  return InstrumentationRuleType.UNKNOWN_TYPE;
};
