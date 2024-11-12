import { Dispatch, SetStateAction } from 'react';
import { DynamicField, ExportedSignals } from '@/types';

export function useEditDestinationFormHandlers(
  setExportedSignals: Dispatch<SetStateAction<ExportedSignals>>,
  setDynamicFields: Dispatch<SetStateAction<DynamicField[]>>
) {
  const handleSignalChange = (
    signal: keyof ExportedSignals,
    value: boolean
  ) => {
    setExportedSignals((prev) => ({ ...prev, [signal]: value }));
  };

  const handleDynamicFieldChange = (name: string, value: any) => {
    setDynamicFields((prev) =>
      prev.map((field) => (field.name === name ? { ...field, value } : field))
    );
  };

  return { handleSignalChange, handleDynamicFieldChange };
}
