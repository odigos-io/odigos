import React, { forwardRef, useImperativeHandle, useMemo } from 'react';
import styled from 'styled-components';
import { useDrawerStore } from '@/store';
import { CardDetails } from '@/components';
import { useActionFormData } from '@/hooks';
import { ChooseActionBody } from '../choose-action-body';
import type { ActionDataParsed, ActionInput } from '@/types';
import buildCardFromActionSpec from './build-card-from-action-spec';
import { ACTION_OPTIONS } from '../choose-action-modal/action-options';

export type ActionDrawerHandle = {
  getCurrentData: () => ActionInput | null;
};

interface Props {
  isEditing: boolean;
}

const ActionDrawer = forwardRef<ActionDrawerHandle, Props>(({ isEditing }, ref) => {
  const selectedItem = useDrawerStore(({ selectedItem }) => selectedItem);
  const { formData, handleFormChange, resetFormData, validateForm } = useActionFormData();

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
      ACTION_OPTIONS.find(({ id }) => id === 'sampler')?.items?.find(({ type }) => type === item.type);

    if (!found) return undefined;

    handleFormChange('type', item.type);
    Object.entries(item.spec).forEach(([k, v]) => {
      switch (k) {
        case 'actionName': {
          handleFormChange('name', v);
          break;
        }

        case 'type':
        case 'name':
        case 'notes':
        case 'signals':
        case 'disable':
        case 'details': {
          if (v !== undefined) handleFormChange(k, v);
          break;
        }

        default: {
          if (v !== undefined) handleFormChange('details', JSON.stringify({ [k]: v }));
          break;
        }
      }
    });

    return found;
  }, [selectedItem, isEditing]);

  useImperativeHandle(ref, () => ({
    getCurrentData: () => (validateForm() ? formData : null),
  }));

  return isEditing && thisAction ? (
    <FormContainer>
      <ChooseActionBody isUpdate action={thisAction} formData={formData} handleFormChange={handleFormChange} />
    </FormContainer>
  ) : (
    <CardDetails data={cardData} />
  );
});

ActionDrawer.displayName = 'ActionDrawer';

export { ActionDrawer };

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
