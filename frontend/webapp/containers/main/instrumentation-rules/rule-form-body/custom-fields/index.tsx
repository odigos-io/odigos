import React from 'react';
import PayloadCollection from './payload-collection';
import { type InstrumentationRuleInput, InstrumentationRuleType } from '@/types';

interface RuleCustomFieldsProps {
  ruleType?: InstrumentationRuleType;
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
}

type ComponentProps = {
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
};

type ComponentType = React.FC<ComponentProps> | null;

const componentsMap: Record<InstrumentationRuleType, ComponentType> = {
  [InstrumentationRuleType.PAYLOAD_COLLECTION]: PayloadCollection,
};

const RuleCustomFields: React.FC<RuleCustomFieldsProps> = ({ ruleType, value, setValue }) => {
  if (!ruleType) return null;

  const Component = componentsMap[ruleType];

  return Component ? <Component value={value} setValue={setValue} /> : null;
};

export default RuleCustomFields;
