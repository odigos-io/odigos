import React from 'react';
import theme from '@/styles/theme';
import styled from 'styled-components';
import RuleCustomFields from './custom-fields';
import type { InstrumentationRuleInput } from '@/types';
import type { RuleOption } from '../rule-modal/rule-options';
import { CheckCircledIcon, CrossCircledIcon } from '@/assets';
import { DocsButton, Input, Text, TextArea, SectionTitle, Segment } from '@/reuseable-components';

interface Props {
  isUpdate?: boolean;
  rule: RuleOption;
  formData: InstrumentationRuleInput;
  formErrors: Record<string, string>;
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

export const RuleFormBody: React.FC<Props> = ({ isUpdate, rule, formData, formErrors, handleFormChange }) => {
  return (
    <Container>
      {isUpdate && (
        <div>
          <FieldTitle>Status</FieldTitle>
          <Segment
            options={[
              { icon: CheckCircledIcon, label: 'active', value: false, selectedBgColor: theme.colors.dark_green },
              { icon: CrossCircledIcon, label: 'inactive', value: true, selectedBgColor: theme.colors.darker_red },
            ]}
            selected={formData.disabled}
            setSelected={(bool) => handleFormChange('disabled', bool)}
          />
        </div>
      )}

      {!isUpdate && <SectionTitle title='' description={rule.docsDescription as string} actionButton={<DocsButton endpoint={rule.docsEndpoint} />} />}

      {!isUpdate && (
        <Input
          title='Rule name'
          placeholder='Use a name that describes the rule'
          value={formData['ruleName']}
          onChange={({ target: { value } }) => handleFormChange('ruleName', value)}
          errorMessage={formErrors['ruleName']}
        />
      )}

      <RuleCustomFields ruleType={rule.type} value={formData} setValue={(key, val) => handleFormChange(key, val)} formErrors={formErrors} />

      <TextArea title='Notes' value={formData['notes']} onChange={({ target: { value } }) => handleFormChange('notes', value)} errorMessage={formErrors['notes']} />
    </Container>
  );
};
