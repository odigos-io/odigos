import React, { useEffect, useState } from 'react';
import buildCard from './build-card';
import styled from 'styled-components';
import { ACTION, DATA_CARDS } from '@/utils';
import OverviewDrawer from '../../overview/overview-drawer';
import { DestinationFormBody } from '../destination-form-body';
import { ConditionDetails, DataCard } from '@odigos/ui-components';
import { Destination, useDrawerStore } from '@odigos/ui-containers';
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
  const { entityId, setDrawerEntityId, setDrawerType } = useDrawerStore();
  const { destinations: destinationTypes } = useDestinationTypes();

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);
  const [thisItem, setThisItem] = useState<Destination | undefined>(undefined);

  const { destinations, updateDestination, deleteDestination } = useDestinationCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setDrawerType(null);
        setDrawerEntityId(null);
        resetFormData();
      }
    },
  });

  useEffect(() => {
    const found = destinations?.find((x) => x.id === entityId);
    setThisItem(found);
  }, [destinations, entityId]);

  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem, destinationTypeDetails, dynamicFields, setDynamicFields } = useDestinationFormData({
    destinationType: thisItem?.destinationType?.type,
    preLoadedFields: thisItem?.fields,
    // TODO: supportedSignals: thisDestination?.supportedSignals,
    // currently, the real "supportedSignals" is being used by "destination" passed as prop to "DestinationFormBody"
  });

  useEffect(() => {
    if (!!thisItem) loadFormWithDrawerItem(thisItem);
  }, [thisItem]);

  if (!thisItem) return null;

  const thisOptionType = destinationTypes?.map(({ items }) => items.filter(({ type }) => type === thisItem.destinationType.type)).filter((arr) => !!arr.length)?.[0]?.[0];

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
    loadFormWithDrawerItem(thisItem);
  };

  const handleDelete = () => {
    deleteDestination(entityId as string);
  };

  const handleSave = (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== thisItem.destinationType.displayName ? newTitle : '';
      handleFormChange('name', title);
      updateDestination(entityId as string, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={thisItem.name || thisItem.destinationType.displayName}
      iconSrc={thisItem.destinationType.imageUrl}
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
            destination={thisOptionType}
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
          <ConditionDetails conditions={thisItem.conditions || []} />
          <DataCard title={DATA_CARDS.DESTINATION_DETAILS} data={!!thisItem ? buildCard(thisItem) : []} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
