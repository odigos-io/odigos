import React from 'react';
import { CheckboxList, Input } from '@/reuseable-components';
import {
  DynamicField,
  ExportedSignals,
  SupportedDestinationSignals,
} from '@/types';
import { DynamicConnectDestinationFormFields } from '@/containers/main/destinations/add-destination/dynamic-form-fields';

interface DestinationFormProps {
  destinationName: string;
  dynamicFields: DynamicField[];
  exportedSignals: ExportedSignals;
  supportedSignals: SupportedDestinationSignals;
  setDestinationName: (name: string) => void;
  handleDynamicFieldChange: (name: string, value: any) => void;
  handleSignalChange: (signal: keyof ExportedSignals, value: boolean) => void;
}

export const DestinationForm: React.FC<DestinationFormProps> = ({
  dynamicFields,
  destinationName,
  exportedSignals,
  supportedSignals,
  setDestinationName,
  handleSignalChange,
  handleDynamicFieldChange,
}) => {
  const monitors = [
    supportedSignals.logs.supported && { id: 'logs', title: 'Logs' },
    supportedSignals.metrics.supported && { id: 'metrics', title: 'Metrics' },
    supportedSignals.traces.supported && { id: 'traces', title: 'Traces' },
  ].filter(Boolean);

  return (
    <>
      <Input
        title="Destination Name"
        placeholder="Enter destination name"
        value={destinationName}
        onChange={(e) => setDestinationName(e.target.value)}
      />
      <CheckboxList
        monitors={monitors as []}
        title="This connection will monitor:"
        exportedSignals={exportedSignals}
        handleSignalChange={handleSignalChange}
      />
      <DynamicConnectDestinationFormFields
        fields={dynamicFields}
        onChange={handleDynamicFieldChange}
      />
    </>
  );
};
