import React from 'react';
import styled from 'styled-components';
import RuleCustomFields from './custom-fields';
import type { InstrumentationRuleInput } from '@/types';
import type { RuleOption } from '../add-rule-modal/rule-options';
import { DocsButton, Input, Text, TextArea, SectionTitle, ToggleButtons } from '@/reuseable-components';

const FieldWrapper = styled.div`
  width: 100%;
  margin: 24px 0;
`;

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`;

interface Props {
  isUpdate?: boolean;
  rule: RuleOption;
  formData: InstrumentationRuleInput;
  handleFormChange: (key: keyof InstrumentationRuleInput, val: any) => void;
}

const ChooseRuleBody: React.FC<Props> = ({ isUpdate, rule, formData, handleFormChange }) => {
  return (
    <>
      {isUpdate && (
        <FieldWrapper>
          <FieldTitle>Status</FieldTitle>
          <ToggleButtons initialValue={!formData.disabled} onChange={(bool) => handleFormChange('disabled', !bool)} />
        </FieldWrapper>
      )}

      {!isUpdate && <SectionTitle title='' description={rule.docsDescription as string} actionButton={<DocsButton endpoint={rule.docsEndpoint} />} />}

      {!isUpdate && (
        <FieldWrapper>
          <Input title='Rule name' placeholder='Use a name that describes the rule' value={formData.ruleName} onChange={({ target: { value } }) => handleFormChange('ruleName', value)} />
        </FieldWrapper>
      )}

      <RuleCustomFields ruleType={rule.type} value={formData} setValue={(key, val) => handleFormChange(key, val)} />

      <FieldWrapper>
        <TextArea title='Notes' value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
      </FieldWrapper>
    </>
  );
};

export { ChooseRuleBody };
