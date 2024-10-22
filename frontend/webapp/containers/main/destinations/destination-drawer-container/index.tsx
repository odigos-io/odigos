import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useState,
} from 'react';
import styled from 'styled-components';
import { ExportedSignals } from '@/types';
import { CardDetails, EditDestinationForm } from '@/components';
import {
  useDestinationFormData,
  useEditDestinationFormHandlers,
} from '@/hooks';

export type DestinationDrawerHandle = {
  getCurrentData: () => {
    type: string;
    exportedSignals: ExportedSignals;
    fields: { key: string; value: any }[];
  };
};

interface DestinationDrawerProps {
  isEditing: boolean;
}

const DestinationDrawer = forwardRef<
  DestinationDrawerHandle,
  DestinationDrawerProps
>(({ isEditing }, ref) => {
  const [isFormDirty, setIsFormDirty] = useState(false);
  const {
    cardData,
    dynamicFields,
    exportedSignals,
    supportedSignals,
    destinationType,
    resetFormData,
    setDynamicFields,
    setExportedSignals,
  } = useDestinationFormData();

  const { handleSignalChange, handleDynamicFieldChange } =
    useEditDestinationFormHandlers(setExportedSignals, setDynamicFields);

  useEffect(() => {
    if (!isEditing && isFormDirty) {
      setIsFormDirty(false);
      resetFormData();
    }
  }, [isEditing]);

  const onDynamicFieldChange = (name: string, value: any) => {
    handleDynamicFieldChange(name, value);
    setIsFormDirty(true);
  };

  const onSignalChange = (signal: keyof ExportedSignals, value: boolean) => {
    handleSignalChange(signal, value);
    setIsFormDirty(true);
  };

  useImperativeHandle(ref, () => ({
    getCurrentData: () => ({
      type: destinationType,
      exportedSignals,
      fields: dynamicFields.map(({ name, value }) => ({ key: name, value })),
    }),
  }));

  return isEditing ? (
    <FormContainer>
      <EditDestinationForm
        dynamicFields={dynamicFields}
        exportedSignals={exportedSignals}
        supportedSignals={supportedSignals}
        handleSignalChange={onSignalChange}
        handleDynamicFieldChange={onDynamicFieldChange}
      />
    </FormContainer>
  ) : (
    <CardDetails data={cardData} />
  );
});

export { DestinationDrawer };

const FormContainer = styled.div`
  display: flex;
  width: 100%;
  flex-direction: column;
  gap: 24px;
  height: 100%;
  overflow-y: auto;
  padding-right: 16px;
  box-sizing: border-box;
  overflow: overlay;
  max-height: calc(100vh - 220px);
`;
