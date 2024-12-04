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
  formErrors: Record<string, string>;
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

export const ActionFormBody: React.FC<Props> = ({ isUpdate, action, formData, formErrors, handleFormChange }) => {
  return (
    <Container>
      {isUpdate && (
        <div>
          <FieldTitle>Status</FieldTitle>
          <ToggleButtons initialValue={!formData.disable} onChange={(bool) => handleFormChange('disable', !bool)} />
        </div>
      )}

      {!isUpdate && <SectionTitle title='' description={action.docsDescription as string} actionButton={<DocsButton endpoint={action.docsEndpoint} />} />}

      <MonitoringCheckboxes
        title='Signals for Processing'
        required
        allowedSignals={action.allowedSignals}
        selectedSignals={formData['signals']}
        setSelectedSignals={(value) => handleFormChange('signals', value)}
        errorMessage={formErrors['signals']}
      />

      {!isUpdate && (
        <Input
          title='Action name'
          placeholder='Use a name that describes the action'
          value={formData['name']}
          onChange={({ target: { value } }) => handleFormChange('name', value)}
          errorMessage={formErrors['name']}
        />
      )}

      <ActionCustomFields actionType={action.type} value={formData['details']} setValue={(val) => handleFormChange('details', val)} errorMessage={formErrors['details']} />

      <TextArea title='Notes' value={formData['notes']} onChange={({ target: { value } }) => handleFormChange('notes', value)} errorMessage={formErrors['notes']} />
    </Container>
  );
};
