import React from 'react';
import styled from 'styled-components';
import RuleCustomFields from './custom-fields';
import type { InstrumentationRuleInput } from '@/types';
import type { RuleOption } from '../add-rule-modal/rule-options';
import { DocsButton, Input, Text, TextArea, SectionTitle, ToggleButtons } from '@/reuseable-components';

interface Props {
  isUpdate?: boolean;
  rule: RuleOption;
  formData: InstrumentationRuleInput;
  handleFormChange: (key: keyof InstrumentationRuleInput, val: any) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 4px;
`;

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`;

const ChooseRuleBody: React.FC<Props> = ({ isUpdate, rule, formData, handleFormChange }) => {
  return (
    <Container>
      {isUpdate ? (
        <div>
          <FieldTitle>Status</FieldTitle>
          <ToggleButtons initialValue={!formData.disabled} onChange={(bool) => handleFormChange('disabled', !bool)} />
        </div>
      ) : (
        <SectionTitle title='' description={rule.docsDescription as string} actionButton={<DocsButton endpoint={rule.docsEndpoint} />} />
      )}

      <Input title='Rule name' placeholder='Use a name that describes the rule' value={formData.ruleName} onChange={({ target: { value } }) => handleFormChange('ruleName', value)} />
      <RuleCustomFields ruleType={rule.type} value={formData} setValue={(key, val) => handleFormChange(key, val)} />
      <TextArea title='Notes' value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
    </Container>
  );
};

export { ChooseRuleBody };
