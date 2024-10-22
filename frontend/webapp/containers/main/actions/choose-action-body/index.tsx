import React from 'react';
import styled from 'styled-components';
import ActionCustomFields from './custom-fields';
import { ActionFormData } from '@/hooks/actions/useActionFormData';
import { type ActionOption } from '../choose-action-modal/action-options';
import { DocsButton, Input, Text, TextArea } from '@/reuseable-components';
import { MonitoringCheckboxes } from '@/reuseable-components/monitoring-checkboxes';

const Description = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  line-height: 150%;
  display: flex;
`;

const FieldWrapper = styled.div`
  width: 100%;
  margin: 8px 0;
`;

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`;

interface ChooseActionContentProps {
  action: ActionOption;
  formData: ActionFormData;
  handleFormChange: (key: keyof ActionFormData, val: any) => void;
}

const ChooseActionBody: React.FC<ChooseActionContentProps> = ({ action, formData, handleFormChange }) => {
  return (
    <>
      <Description>
        {action.docsDescription}
        <DocsButton endpoint={action.docsEndpoint} />
      </Description>

      <FieldWrapper>
        <MonitoringCheckboxes
          allowedSignals={action.allowedSignals}
          selectedSignals={formData.signals}
          setSelectedSignals={(value) => handleFormChange('signals', value)}
        />
      </FieldWrapper>

      <FieldWrapper>
        <FieldTitle>Action name</FieldTitle>
        <Input
          placeholder='Use a name that describes the action'
          value={formData.name}
          onChange={({ target: { value } }) => handleFormChange('name', value)}
        />
      </FieldWrapper>

      <ActionCustomFields actionType={action.type} value={formData.details} setValue={(val) => handleFormChange('details', val)} />

      <FieldWrapper>
        <FieldTitle>Notes</FieldTitle>
        <TextArea value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
      </FieldWrapper>
    </>
  );
};

export { ChooseActionBody };
