import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import { ActionFormBody } from '../';
import styled from 'styled-components';
import { useDrawerStore } from '@odigos/ui-containers';
import { useActionCRUD, useActionFormData } from '@/hooks';
import { ConditionDetails, DataCard } from '@odigos/ui-components';
import { ACTION_OPTIONS, CRUD, DISPLAY_TITLES, getActionIcon } from '@odigos/ui-utils';
import OverviewDrawer from '../../overview/overview-drawer';

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
  const { drawerEntityId, setDrawerEntityId, setDrawerType } = useDrawerStore();

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useActionFormData();
  const { actions, updateAction, deleteAction } = useActionCRUD({
    onSuccess: (type) => {
      setIsEditing(false);
      setIsFormDirty(false);

      if (type === CRUD.DELETE) {
        setDrawerType(null);
        setDrawerEntityId(null);
        resetFormData();
      }
    },
  });

  const thisItem = useMemo(() => {
    const found = actions?.find((x) => x.id === drawerEntityId);
    if (!!found) loadFormWithDrawerItem(found);

    return found;
  }, [actions, drawerEntityId]);

  if (!thisItem) return null;

  const thisOptionType =
    ACTION_OPTIONS.find(({ type }) => type === thisItem.type) ||
    ACTION_OPTIONS.find(({ id }) => id === 'attributes')?.items?.find(({ type }) => type === thisItem.type) ||
    ACTION_OPTIONS.find(({ id }) => id === 'sampler')?.items?.find(({ type }) => type === thisItem.type);

  const handleEdit = (bool?: boolean) => {
    setIsEditing(typeof bool === 'boolean' ? bool : true);
  };

  const handleCancel = () => {
    setIsEditing(false);
    setIsFormDirty(false);
    loadFormWithDrawerItem(thisItem);
  };

  const handleDelete = () => {
    deleteAction(drawerEntityId as string, thisItem.type);
  };

  const handleSave = (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: CRUD.UPDATE })) {
      const title = newTitle !== thisItem.type ? newTitle : '';
      handleFormChange('name', title);
      updateAction(drawerEntityId as string, { ...formData, name: title });
    }
  };

  return (
    <OverviewDrawer
      title={thisItem.spec.actionName || thisItem.type}
      icon={getActionIcon(thisItem.type)}
      isEdit={isEditing}
      isFormDirty={isFormDirty}
      onEdit={handleEdit}
      onSave={handleSave}
      onDelete={handleDelete}
      onCancel={handleCancel}
    >
      {isEditing && thisOptionType ? (
        <FormContainer>
          <ActionFormBody
            isUpdate
            action={thisOptionType}
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
          <ConditionDetails conditions={thisItem.conditions || []} />
          <DataCard title={DISPLAY_TITLES.ACTION_DETAILS} data={!!thisItem ? buildCard(thisItem) : []} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
