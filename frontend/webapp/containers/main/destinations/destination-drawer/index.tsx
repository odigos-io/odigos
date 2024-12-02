import React, { useMemo, useState } from 'react';
import { ACTION } from '@/utils';
import buildCard from './build-card';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import buildDrawerItem from './build-drawer-item';
import OverviewDrawer from '../../overview/overview-drawer';
import { DestinationFormBody } from '../destination-form-body';
import { ConditionDetails, DataCard } from '@/reuseable-components';
import { OVERVIEW_ENTITY_TYPES, type ActualDestination } from '@/types';
import { useDestinationCRUD, useDestinationFormData, useDestinationTypes } from '@/hooks';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

const DataContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
`;

export const DestinationDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { destinations: destinationTypes } = useDestinationTypes();

  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem, destinationTypeDetails, dynamicFields, setDynamicFields } = useDestinationFormData({
    destinationType: (selectedItem?.item as ActualDestination)?.destinationType?.type,
    preLoadedFields: (selectedItem?.item as ActualDestination)?.fields,
    // TODO: supportedSignals: thisDestination?.supportedSignals,
    // currently, the real "supportedSignals" is being used by "destination" passed as prop to "DestinationFormBody"
  });

  const { updateDestination, deleteDestination } = useDestinationCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setSelectedItem(null);
      } else {
        const { item } = selectedItem as { item: ActualDestination };
        const { id } = item;
        setSelectedItem({ id, type: OVERVIEW_ENTITY_TYPES.DESTINATION, item: buildDrawerItem(id, formData, item) });
      }
    },
  });

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const cardData = useMemo(() => {
    if (!selectedItem || !destinationTypeDetails) return [];

    const { item } = selectedItem as { item: ActualDestination };
    const arr = buildCard(item, destinationTypeDetails);

    return arr;
  }, [selectedItem, destinationTypeDetails]);

  const thisDestination = useMemo(() => {
    if (!destinationTypes.length || !selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: ActualDestination };
    const found = destinationTypes.map(({ items }) => items.filter(({ type }) => type === item.destinationType.type)).filter((arr) => !!arr.length)[0][0];

    if (!found) return undefined;

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [destinationTypes, selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: string; item: ActualDestination };

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    await deleteDestination(id);
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== item.destinationType.displayName ? newTitle : '';
      handleFormChange('name', title);
      await updateDestination(id, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={item.name || item.destinationType.displayName}
      imageUri={item.destinationType.imageUrl}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing ? (
        <FormContainer>
          <DestinationFormBody
            isUpdate
            destination={thisDestination}
            formData={formData}
            formErrors={formErrors}
            validateForm={validateForm}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
            dynamicFields={dynamicFields}
            setDynamicFields={(...params) => {
              setIsFormDirty(true);
              setDynamicFields(...params);
            }}
          />
        </FormContainer>
      ) : (
        <DataContainer>
          <ConditionDetails conditions={item.conditions} />
          <DataCard title='Destination Details' data={cardData} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
