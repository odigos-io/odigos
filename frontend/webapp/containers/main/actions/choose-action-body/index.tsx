import React from 'react';
import styled from 'styled-components';
import { type ActionInput } from '@/types';
import ActionCustomFields from './custom-fields';
import { type ActionOption } from '../choose-action-modal/action-options';
import { DocsButton, Input, Text, TextArea, MonitoringCheckboxes, SectionTitle, ToggleButtons } from '@/reuseable-components';

const FieldWrapper = styled.div`
  width: calc(100% - 16px);
  margin: 24px 0;
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
          <ToggleButtons initialValue={!formData.disable} onChange={(bool) => handleFormChange('disable', !bool)} />
        </FieldWrapper>
      )}

      {!isUpdate && <SectionTitle title='' description={action.docsDescription as string} actionButton={<DocsButton endpoint={action.docsEndpoint} />} />}

      <FieldWrapper>
        <MonitoringCheckboxes allowedSignals={action.allowedSignals} selectedSignals={formData.signals} setSelectedSignals={(value) => handleFormChange('signals', value)} />
      </FieldWrapper>

      {!isUpdate && (
        <FieldWrapper>
          <Input title='Action name' placeholder='Use a name that describes the action' value={formData.name} onChange={({ target: { value } }) => handleFormChange('name', value)} />
        </FieldWrapper>
      )}

      <ActionCustomFields actionType={action.type} value={formData.details} setValue={(val) => handleFormChange('details', val)} />

      <FieldWrapper>
        <TextArea title='Notes' value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
      </FieldWrapper>
    </>
  );
};

export { ChooseActionBody };
