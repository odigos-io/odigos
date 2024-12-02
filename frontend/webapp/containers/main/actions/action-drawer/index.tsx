import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import { ActionFormBody } from '../';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { ACTION, getActionIcon } from '@/utils';
import buildDrawerItem from './build-drawer-item';
import { DataCard } from '@/reuseable-components';
import { useActionCRUD, useActionFormData } from '@/hooks';
import OverviewDrawer from '../../overview/overview-drawer';
import { ACTION_OPTIONS } from '../action-modal/action-options';
import { OVERVIEW_ENTITY_TYPES, type ActionDataParsed } from '@/types';

interface Props {}

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;

export const ActionDrawer: React.FC<Props> = () => {
  const { selectedItem, setSelectedItem } = useDrawerStore();
  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useActionFormData();

  const { updateAction, deleteAction } = useActionCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === ACTION.DELETE) {
        setSelectedItem(null);
      } else {
        const { item } = selectedItem as { item: ActionDataParsed };
        const { id } = item;
        setSelectedItem({ id, type: OVERVIEW_ENTITY_TYPES.ACTION, item: buildDrawerItem(id, formData, item) });
      }
    },
  });

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

    if (!found) return undefined;

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
      imageUri={getActionIcon(item.type)}
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
        <DataCard title='Action Details' data={cardData} />
      )}
    </OverviewDrawer>
  );
};
