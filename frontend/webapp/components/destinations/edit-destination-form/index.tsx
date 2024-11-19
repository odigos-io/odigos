import React from 'react';
import styled from 'styled-components';
import { CheckboxList } from '@/reuseable-components';
import type { DynamicField, ExportedSignals, SupportedDestinationSignals } from '@/types';
import { DynamicConnectDestinationFormFields } from '@/containers/main/destinations/add-destination/dynamic-form-fields';

interface DestinationFormProps {
  exportedSignals: ExportedSignals;
  supportedSignals: SupportedDestinationSignals;
  dynamicFields: DynamicField[];
  handleDynamicFieldChange: (name: string, value: any) => void;
  handleSignalChange: (signal: keyof ExportedSignals, value: boolean) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 4px;
`;

export const EditDestinationForm: React.FC<DestinationFormProps> = ({ exportedSignals, supportedSignals, dynamicFields, handleSignalChange, handleDynamicFieldChange }) => {
  const monitors = [
    supportedSignals.logs.supported && { id: 'logs', title: 'Logs' },
    supportedSignals.metrics.supported && { id: 'metrics', title: 'Metrics' },
    supportedSignals.traces.supported && { id: 'traces', title: 'Traces' },
  ].filter(Boolean);

  return (
    <Container>
      <CheckboxList monitors={monitors as []} title='This connection will monitor:' exportedSignals={exportedSignals} handleSignalChange={handleSignalChange} />
      <DynamicConnectDestinationFormFields fields={dynamicFields} onChange={handleDynamicFieldChange} />
    </Container>
  );
};
