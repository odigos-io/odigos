import React from 'react';
import { CheckboxList } from '@/reuseable-components';
import { DynamicConnectDestinationFormFields } from '@/containers/main/destinations/add-destination/dynamic-form-fields';
import {
  DynamicField,
  ExportedSignals,
  SupportedDestinationSignals,
} from '@/types';

interface DestinationFormProps {
  dynamicFields: DynamicField[];
  exportedSignals: ExportedSignals;
  supportedSignals: SupportedDestinationSignals;
  handleDynamicFieldChange: (name: string, value: any) => void;
  handleSignalChange: (signal: keyof ExportedSignals, value: boolean) => void;
}

export const EditDestinationForm: React.FC<DestinationFormProps> = ({
  dynamicFields,
  exportedSignals,
  supportedSignals,
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
