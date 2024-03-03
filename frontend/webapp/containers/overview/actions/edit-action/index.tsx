'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { useActionState } from '@/hooks';
import { useSearchParams } from 'next/navigation';
import { ACTION, ACTIONS, ACTION_DOCS_LINK } from '@/utils';
import { MultiCheckboxComponent, DynamicActionForm } from '@/components';
import {
  KeyvalButton,
  KeyvalInput,
  KeyvalLink,
  KeyvalLoader,
  KeyvalText,
  KeyvalTextArea,
} from '@/design.system';
import {
  CreateActionWrapper,
  CreateButtonWrapper,
  DescriptionWrapper,
  KeyvalInputWrapper,
  LoaderWrapper,
  TextareaWrapper,
} from './styled';

const ACTION_ID = 'id';

export function EditActionContainer(): React.JSX.Element {
  const [currentActionType, setCurrentActionType] = useState<string>();

  const {
    actionState,
    onChangeActionState,
    updateCurrentAction,
    buildActionData,
  } = useActionState();

  const { actionName, actionNote, actionData, selectedMonitors } = actionState;

  const search = useSearchParams();

  useEffect(() => {
    const actionId = search.get(ACTION_ID);
    if (!actionId) return;
    setCurrentActionType('add-cluster-info');
    buildActionData(actionId);
  }, [search]);

  if (!actionState || !currentActionType)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <CreateActionWrapper>
      <DescriptionWrapper>
        <KeyvalText size={14}>
          {ACTIONS[currentActionType].DESCRIPTION}
        </KeyvalText>
        <KeyvalLink
          value={ACTION.LINK_TO_DOCS}
          fontSize={14}
          onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
        />
      </DescriptionWrapper>
      <MultiCheckboxComponent
        title={ACTIONS.MONITORS_TITLE}
        checkboxes={selectedMonitors}
        onSelectionChange={(newMonitors) =>
          onChangeActionState('selectedMonitors', newMonitors)
        }
      />
      <KeyvalInputWrapper>
        <KeyvalInput
          label={ACTIONS.ACTION_NAME}
          value={actionName}
          onChange={(name) => onChangeActionState('actionName', name)}
        />
      </KeyvalInputWrapper>
      <DynamicActionForm
        type={currentActionType}
        data={actionData}
        onChange={onChangeActionState}
      />
      <TextareaWrapper>
        <KeyvalTextArea
          label={ACTIONS.ACTION_NOTE}
          value={actionNote}
          placeholder={ACTIONS.NOTE_PLACEHOLDER}
          onChange={(e) => onChangeActionState('actionNote', e.target.value)}
        />
      </TextareaWrapper>
      <CreateButtonWrapper>
        <KeyvalButton onClick={updateCurrentAction} disabled={!actionData}>
          <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
            {ACTIONS.UPDATE_ACTION}
          </KeyvalText>
        </KeyvalButton>
      </CreateButtonWrapper>
    </CreateActionWrapper>
  );
}
