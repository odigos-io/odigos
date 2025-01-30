import React from 'react';
import { Types } from '@odigos/ui-components';
import CodeAttributes from './code-attributes';
import PayloadCollection from './payload-collection';
import { type InstrumentationRuleInput } from '@/types';

interface Props {
  ruleType?: Types.INSTRUMENTATION_RULE_TYPE;
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

const componentsMap: Record<Types.INSTRUMENTATION_RULE_TYPE, ComponentType> = {
  [Types.INSTRUMENTATION_RULE_TYPE.PAYLOAD_COLLECTION]: PayloadCollection,
  [Types.INSTRUMENTATION_RULE_TYPE.CODE_ATTRIBUTES]: CodeAttributes,
  [Types.INSTRUMENTATION_RULE_TYPE.UNKNOWN_TYPE]: null,
};

const RuleCustomFields: React.FC<Props> = ({ ruleType, value, setValue, formErrors }) => {
  if (!ruleType) return null;

  const Component = componentsMap[ruleType];

  return Component ? <Component value={value} setValue={setValue} formErrors={formErrors} /> : null;
};

export default RuleCustomFields;
