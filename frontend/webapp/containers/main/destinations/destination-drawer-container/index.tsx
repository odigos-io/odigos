import React, { useState } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { ActualDestination } from '@/types';
import OverviewDrawer from '../../overview/overview-drawer';
import { CardDetails, EditDestinationForm } from '@/components';
import { useDestinationCRUD, useDestinationFormData, useEditDestinationFormHandlers } from '@/hooks';

interface Props {}

const DestinationDrawer: React.FC<Props> = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { cardData, dynamicFields, exportedSignals, supportedSignals, destinationType, resetFormData, setDynamicFields, setExportedSignals } =
    useDestinationFormData();
  const { handleSignalChange, handleDynamicFieldChange } = useEditDestinationFormHandlers(setExportedSignals, setDynamicFields);
  const { updateDestination, deleteDestination } = useDestinationCRUD();

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem;

  const handleEdit = (bool?: boolean) => {
    if (typeof bool === 'boolean') {
      setIsEditing(bool);
    } else {
      setIsEditing(true);
    }
  };

  const handleCancel = () => {
    resetFormData();
    setIsEditing(false);
  };

  const handleDelete = async () => {
    await deleteDestination(id as string);
  };

  const handleSave = async (newTitle: string) => {
    const payload = {
      type: destinationType,
      name: newTitle,
      exportedSignals,
      fields: dynamicFields.map(({ name, value }) => ({ key: name, value })),
    };

    await updateDestination(id as string, payload);
  };

  return (
    <OverviewDrawer
      title={(item as ActualDestination).name}
      imageUri={(item as ActualDestination).destinationType.imageUrl}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing ? (
        <FormContainer>
          <EditDestinationForm
            dynamicFields={dynamicFields}
            exportedSignals={exportedSignals}
            supportedSignals={supportedSignals}
            handleSignalChange={(...params) => {
              setIsFormDirty(true);
              handleSignalChange(...params);
            }}
            handleDynamicFieldChange={(...params) => {
              setIsFormDirty(true);
              handleDynamicFieldChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <CardDetails data={cardData} />
      )}
    </OverviewDrawer>
  );
};

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
