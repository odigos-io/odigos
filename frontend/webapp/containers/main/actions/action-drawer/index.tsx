import React, { useMemo, useState } from 'react';
import buildCard from './build-card';
import { ActionFormBody } from '../';
import styled from 'styled-components';
import { ACTION, DATA_CARDS } from '@/utils';
import { useDrawerStore } from '@odigos/ui-containers';
import { useActionCRUD, useActionFormData } from '@/hooks';
import OverviewDrawer from '../../overview/overview-drawer';
import { ACTION_OPTIONS, getActionIcon } from '@odigos/ui-utils';
import { ConditionDetails, DataCard } from '@odigos/ui-components';

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
  const { entityId, setDrawerEntityId, setDrawerType } = useDrawerStore();

  const [isEditing, setIsEditing] = useState(false);
  const [isFormDirty, setIsFormDirty] = useState(false);

  const { formData, formErrors, handleFormChange, resetFormData, validateForm, loadFormWithDrawerItem } = useActionFormData();
  const { actions, updateAction, deleteAction } = useActionCRUD({
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

  const thisItem = useMemo(() => {
    const found = actions?.find((x) => x.id === entityId);
    if (!!found) loadFormWithDrawerItem(found);

    return found;
  }, [actions, entityId]);

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
    deleteAction(entityId as string, thisItem.type);
  };

  const handleSave = (newTitle: string) => {
    if (validateForm({ withAlert: true, alertTitle: ACTION.UPDATE })) {
      const title = newTitle !== thisItem.type ? newTitle : '';
      handleFormChange('name', title);
      updateAction(entityId as string, { ...formData, name: title });
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
          <DataCard title={DATA_CARDS.ACTION_DETAILS} data={!!thisItem ? buildCard(thisItem) : []} />
        </DataContainer>
      )}
    </OverviewDrawer>
  );
};
