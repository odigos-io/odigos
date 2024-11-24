import React, { useMemo, useState } from 'react';
import styled from 'styled-components';
import { getActionIcon } from '@/utils';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import type { ActionDataParsed } from '@/types';
import { ChooseActionBody } from '../choose-action-body';
import { useActionCRUD, useActionFormData } from '@/hooks';
import OverviewDrawer from '../../overview/overview-drawer';
import buildCardFromActionSpec from './build-card-from-action-spec';
import { ACTION_OPTIONS } from '../choose-action-modal/action-options';

interface Props {}

const ActionDrawer: React.FC<Props> = () => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { formData, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useActionFormData();
  const { updateAction, deleteAction } = useActionCRUD();

  const cardData = useMemo(() => {
    if (!selectedItem) return [];

    const { item } = selectedItem as { item: ActionDataParsed };
    const arr = buildCardFromActionSpec(item);

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
    await deleteAction(id as string, (item as ActionDataParsed).type);
  };

  const handleSave = async (newTitle: string) => {
    if (validateForm({ withAlert: true })) {
      const title = newTitle !== (item as ActionDataParsed).type ? newTitle : '';

      await updateAction(id as string, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={(item as ActionDataParsed).spec.actionName || (item as ActionDataParsed).type}
      imageUri={getActionIcon((item as ActionDataParsed).type)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing && thisAction ? (
        <FormContainer>
          <ChooseActionBody
            isUpdate
            action={thisAction}
            formData={formData}
            handleFormChange={(...params) => {
              setIsFormDirty(true);
              handleFormChange(...params);
            }}
          />
        </FormContainer>
      ) : (
        <CardDetails data={cardData} />
      )}
    </OverviewDrawer>
  );
};

export { ActionDrawer };

const FormContainer = styled.div`
  width: 100%;
  height: 100%;
  max-height: calc(100vh - 220px);
  overflow: overlay;
  overflow-y: auto;
`;
