import React, { useEffect, useMemo, useState } from 'react';
import buildCard from './build-card';
import { ActionFormBody } from '../';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { ACTION, DATA_CARDS } from '@/utils';
import { type ActionDataParsed } from '@/types';
import buildDrawerItem from './build-drawer-item';
import { useActionCRUD, useActionFormData } from '@/hooks';
import OverviewDrawer from '../../overview/overview-drawer';
import { ConditionDetails, DataCard } from '@/reuseable-components';
import { ACTION_OPTIONS, getActionIcon, Types } from '@odigos/ui-components';

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

export const ActionDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useActionFormData();
  const { actions, updateAction, deleteAction } = useActionCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) setSelectedItem(null);
      else reSelectItem();
    },
  });

  const reSelectItem = (fetchedItems?: typeof actions) => {
    const { item } = selectedItem as { item: ActionDataParsed };
    const { id } = item;

    if (!!fetchedItems?.length) {
      const found = fetchedItems.find((x) => x.id === id);
      if (!!found) {
        return setSelectedItem({ id, type: Types.ENTITY_TYPES.ACTION, item: found });
      }
    }

    setSelectedItem({ id, type: Types.ENTITY_TYPES.ACTION, item: buildDrawerItem(id, formData, item) });
  };

  // This should keep the drawer up-to-date with the latest data
  useEffect(() => reSelectItem(actions), [actions]);

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: ActionDataParsed };
    const arr = buildCard(item);

    return arr;
  }, [selectedItem]);

  const thisAction = useMemo(() => {
    if (!selectedItem || !isEditing) {
      resetFormData();
      return undefined;
    }

    const { item } = selectedItem as { item: ActionDataParsed };
    const found =
      ACTION_OPTIONS.find(({ type }) => type === item.type) ||
      ACTION_OPTIONS.find(({ id }) => id === 'attributes')?.items?.find(({ type }) => type === item.type) ||
      ACTION_OPTIONS.find(({ id }) => id === 'sampler')?.items?.find(({ type }) => type === item.type);

    loadFormWithDrawerItem(selectedItem);

    return found;
  }, [selectedItem, isEditing]);

  if (!selectedItem?.item) return null;
  const { id, item } = selectedItem as { id: string; item: ActionDataParsed };

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
  };

  const handleDelete = async () => {
    await deleteAction(id, item.type);
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== item.type ? newTitle : '';
      handleFormChange('name', title);
      await updateAction(id, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={item.spec.actionName || item.type}
      icon={getActionIcon(item.type)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing && thisAction ? (
        <FormContainer>
          <ActionFormBody
            isUpdate
            action={thisAction}
            formData={formData}
            formErrors={formErrors}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <DataContainer>
          <ConditionDetails conditions={item?.conditions || []} />
          <DataCard title={DATA_CARDS.ACTION_DETAILS} data={cardData} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
