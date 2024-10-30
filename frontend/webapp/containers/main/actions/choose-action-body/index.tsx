import React from 'react';
import styled from 'styled-components';
import { type ActionInput } from '@/types';
import ActionCustomFields from './custom-fields';
import { type ActionOption } from '../choose-action-modal/action-options';
import { DocsButton, Input, Text, TextArea, Toggle, MonitoringCheckboxes, SectionTitle } from '@/reuseable-components';

const FieldWrapper = styled.div`
  width: 100%;
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
          <Toggle
            title={!formData.disable ? 'Active' : 'Inactive'}
            initialValue={!formData.disable}
            onChange={(bool) => handleFormChange('disable', !bool)}
          />
        </FieldWrapper>
      )}

      {!isUpdate && (
        <SectionTitle title='' description={action.docsDescription as string} actionButton={<DocsButton endpoint={action.docsEndpoint} />} />
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
          <Input
            title='Action name'
            placeholder='Use a name that describes the action'
            value={formData.name}
            onChange={({ target: { value } }) => handleFormChange('name', value)}
          />
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

/* <CheckboxList
        monitors={[
          { id: 'logs', title: 'Logs' },
          { id: 'traces', title: 'Traces' },
          { id: 'metrics', title: 'Metrics' },
        ]}
        title='Monitoring'
        exportedSignals={{
          logs: formData.signals.includes('LOGS'),
          traces: formData.signals.includes('TRACES'),
          metrics: formData.signals.includes('METRICS'),
        }}
        handleSignalChange={(key, isAdd) => {
          const selectedSignals = [];
          const keyUpper = (key as string).toUpperCase() as SignalUppercase;
          const payload = isAdd ? [...selectedSignals, keyUpper] : selectedSignals.filter((str) => str !== keyUpper);

          handleFormChange('signals', payload);
        }}
      /> */
