import React, { useEffect, useMemo, useState } from 'react';
import buildCard from './build-card';
import styled from 'styled-components';
import { ACTION, DATA_CARDS } from '@/utils';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import buildDrawerItem from './build-drawer-item';
import OverviewDrawer from '../../overview/overview-drawer';
import { DestinationFormBody } from '../destination-form-body';
import { ConditionDetails, DataCard } from '@odigos/ui-components';
import { type Destination, useDrawerStore } from '@odigos/ui-containers';
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
    destinationType: (selectedItem?.item as Destination)?.destinationType?.type,
    preLoadedFields: (selectedItem?.item as Destination)?.fields,
    // TODO: supportedSignals: thisDestination?.supportedSignals,
    // currently, the real "supportedSignals" is being used by "destination" passed as prop to "DestinationFormBody"
  });

  const { destinations, updateDestination, deleteDestination } = useDestinationCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) setSelectedItem(null);
      else reSelectItem();
    },
  });

  // TODO: check if the item is already set on-mount
  // drawerItem['item'] = destinations.find((item) => item.id === drawerItem['id']);
  const reSelectItem = (fetchedItems?: typeof destinations) => {
    const { item } = selectedItem as { item: Destination };
    const { id } = item;

    if (!!fetchedItems?.length) {
      const found = fetchedItems.find((x) => x.id === id);
      if (!!found) {
        return setSelectedItem({ id, type: ENTITY_TYPES.DESTINATION, item: found });
      }
    }

    setSelectedItem({ id, type: ENTITY_TYPES.DESTINATION, item: buildDrawerItem(id, formData, item) });
  };

  // This should keep the drawer up-to-date with the latest data
  useEffect(() => reSelectItem(destinations), [destinations]);

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: Destination };
    const arr = buildCard(item, destinationTypeDetails);

    return arr;
  }, [selectedItem, destinationTypeDetails]);

  const thisDestinationType = useMemo(() => {
    if (!destinationTypes.length || !selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: Destination };
    const found = destinationTypes.map(({ items }) => items.filter(({ type }) => type === item.destinationType.type)).filter((arr) => !!arr.length)?.[0]?.[0];

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [destinationTypes, selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: string; item: Destination };

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
      iconSrc={item.destinationType.imageUrl}
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
            destination={thisDestinationType}
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
          <ConditionDetails conditions={item.conditions || []} />
          <DataCard title={DATA_CARDS.DESTINATION_DETAILS} data={cardData} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
