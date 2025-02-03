import React from 'react';
import CodeAttributes from './code-attributes';
import PayloadCollection from './payload-collection';
import { type InstrumentationRuleInput } from '@/types';
import { INSTRUMENTATION_RULE_TYPE } from '@odigos/ui-utils';

interface Props {
  ruleType?: INSTRUMENTATION_RULE_TYPE;
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
  formErrors: Record<string, string>;
}

interface ComponentProps {
  value: Props['value'];
  setValue: Props['setValue'];
  formErrors: Props['formErrors'];
}

type ComponentType = React.FC<ComponentProps> | null;

const componentsMap: Record<INSTRUMENTATION_RULE_TYPE, ComponentType> = {
  [INSTRUMENTATION_RULE_TYPE.PAYLOAD_COLLECTION]: PayloadCollection,
  [INSTRUMENTATION_RULE_TYPE.CODE_ATTRIBUTES]: CodeAttributes,
  [INSTRUMENTATION_RULE_TYPE.UNKNOWN_TYPE]: null,
};

const RuleCustomFields: React.FC<Props> = ({ ruleType, value, setValue, formErrors }) => {
  if (!ruleType) return null;

  const Component = componentsMap[ruleType];

  return Component ? <Component value={value} setValue={setValue} formErrors={formErrors} /> : null;
};

export default RuleCustomFields;
