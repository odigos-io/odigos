'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { useActionState } from '@/hooks';
import { useSearchParams } from 'next/navigation';
import { ACTION, ACTIONS, ACTION_ITEM_DOCS_LINK } from '@/utils';
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

const ACTION_TYPE = 'type';

export function CreateActionContainer(): React.JSX.Element {
  const [currentActionType, setCurrentActionType] = useState<string>();
  const { actionState, onChangeActionState, upsertAction } = useActionState();
  const { actionName, actionNote, actionData, selectedMonitors } = actionState;

  const search = useSearchParams();

  useEffect(() => {
    const action = search.get(ACTION_TYPE);
    action && setCurrentActionType(action);
  }, [search]);

  if (!currentActionType)
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
          onClick={() =>
            window.open(
              `${ACTION_ITEM_DOCS_LINK}/${currentActionType.toLowerCase()}`,
              '_blank'
            )
          }
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
        <KeyvalButton onClick={upsertAction} disabled={!actionData}>
          <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
            {ACTIONS.CREATE_ACTION}
          </KeyvalText>
        </KeyvalButton>
      </CreateButtonWrapper>
    </CreateActionWrapper>
  );
}
