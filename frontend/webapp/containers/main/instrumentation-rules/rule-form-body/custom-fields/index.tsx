import React from 'react';
import PayloadCollection from './payload-collection';
import { type InstrumentationRuleInput, InstrumentationRuleType } from '@/types';

interface Props {
  ruleType?: InstrumentationRuleType;
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

const componentsMap: Record<InstrumentationRuleType, ComponentType> = {
  [InstrumentationRuleType.PAYLOAD_COLLECTION]: PayloadCollection,
};

const RuleCustomFields: React.FC<Props> = ({ ruleType, value, setValue, formErrors }) => {
  if (!ruleType) return null;

  const Component = componentsMap[ruleType];

  return Component ? <Component value={value} setValue={setValue} formErrors={formErrors} /> : null;
};

export default RuleCustomFields;
