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

  const { cardData, dynamicFields, destinationName, exportedSignals, supportedSignals, destinationType, resetFormData, setDynamicFields, setDestinationName, setExportedSignals } =
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

  const handleSave = async () => {
    const payload = {
      type: destinationType,
      name: destinationName,
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
            exportedSignals={exportedSignals}
            supportedSignals={supportedSignals}
            destinationName={destinationName}
            dynamicFields={dynamicFields}
            handleSignalChange={(...params) => {
              setIsFormDirty(true);
              handleSignalChange(...params);
            }}
            handleNameChange={(...params) => {
              setIsFormDirty(true);
              setDestinationName(...params);
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
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;
