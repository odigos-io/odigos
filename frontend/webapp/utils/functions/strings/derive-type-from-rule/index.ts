import { Types } from '@odigos/ui-components';
import { type InstrumentationRuleInput, type InstrumentationRuleSpec } from '@/types';

export const deriveTypeFromRule = (rule: InstrumentationRuleInput | InstrumentationRuleSpec) => {
  const allKeysAreNull = (obj: Record<string, any>) => Object.values(obj).every((v) => v === null);

  if (rule.payloadCollection && !allKeysAreNull(rule.payloadCollection)) {
    return Types.INSTRUMENTATION_RULE_TYPE.PAYLOAD_COLLECTION;
  } else if (rule.codeAttributes && !allKeysAreNull(rule.codeAttributes)) {
    return Types.INSTRUMENTATION_RULE_TYPE.CODE_ATTRIBUTES;
  }

  return Types.INSTRUMENTATION_RULE_TYPE.UNKNOWN_TYPE;
};
