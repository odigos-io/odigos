import React from 'react';
import styled from 'styled-components';
import { type ActionInput } from '@/types';
import ActionCustomFields from './custom-fields';
import { type ActionOption } from '../action-modal/action-options';
import { DocsButton, Input, Text, TextArea, MonitoringCheckboxes, SectionTitle, ToggleButtons } from '@/reuseable-components';

interface Props {
  isUpdate?: boolean;
  action: ActionOption;
  formData: ActionInput;
  handleFormChange: (key: keyof ActionInput, val: any) => void;
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

export const ActionFormBody: React.FC<Props> = ({ isUpdate, action, formData, handleFormChange }) => {
  return (
    <Container>
      {isUpdate && (
        <div>
          <FieldTitle>Status</FieldTitle>
          <ToggleButtons initialValue={!formData.disable} onChange={(bool) => handleFormChange('disable', !bool)} />
        </div>
      )}

      {!isUpdate && <SectionTitle title='' description={action.docsDescription as string} actionButton={<DocsButton endpoint={action.docsEndpoint} />} />}

      <MonitoringCheckboxes allowedSignals={action.allowedSignals} selectedSignals={formData.signals} setSelectedSignals={(value) => handleFormChange('signals', value)} />

      {!isUpdate && <Input title='Action name' placeholder='Use a name that describes the action' value={formData.name} onChange={({ target: { value } }) => handleFormChange('name', value)} />}

      <ActionCustomFields actionType={action.type} value={formData.details} setValue={(val) => handleFormChange('details', val)} />

      <TextArea title='Notes' value={formData.notes} onChange={({ target: { value } }) => handleFormChange('notes', value)} />
    </Container>
  );
};
