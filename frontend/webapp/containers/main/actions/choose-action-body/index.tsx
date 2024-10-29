import React from 'react';
import styled from 'styled-components';
import { type ActionInput } from '@/types';
import ActionCustomFields from './custom-fields';
import { type ActionOption } from '../choose-action-modal/action-options';
import { Checkbox, DocsButton, Input, Text, TextArea } from '@/reuseable-components';
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
  isUpdate?: boolean;
  action: ActionOption;
  formData: ActionInput;
  handleFormChange: (key: keyof ActionInput, val: any) => void;
}

const ChooseActionBody: React.FC<ChooseActionContentProps> = ({ isUpdate, action, formData, handleFormChange }) => {
  return (
    <>
      {isUpdate && (
        <FieldWrapper>
          <FieldTitle>Status</FieldTitle>
          <Checkbox title='Active' initialValue={!formData.disable} onChange={(bool) => handleFormChange('disable', !bool)} />
        </FieldWrapper>
      )}

      {!isUpdate && (
        <Description>
          {action.docsDescription}
          <DocsButton endpoint={action.docsEndpoint} />
        </Description>
      )}

      <FieldWrapper>
        <MonitoringCheckboxes
          allowedSignals={action.allowedSignals}
          selectedSignals={formData.signals}
          setSelectedSignals={(value) => handleFormChange('signals', value)}
        />
      </FieldWrapper>

      {!isUpdate && (
        <FieldWrapper>
          <FieldTitle>Action name</FieldTitle>
          <Input
            placeholder='Use a name that describes the action'
            value={formData.name}
            onChange={({ target: { value } }) => handleFormChange('name', value)}
          />
        </FieldWrapper>
      )}

      <ActionCustomFields actionType={action.type} value={formData.details} setValue={(val) => handleFormChange('details', val)} />

      <FieldWrapper>
        <FieldTitle>Notes</FieldTitle>
        <TextArea value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
      </FieldWrapper>
    </>
  );
};

export { ChooseActionBody };
