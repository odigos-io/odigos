'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { ActionsType } from '@/types';
import { useActionState } from '@/hooks';
import { useSearchParams } from 'next/navigation';
import { ACTION, ACTIONS, ACTION_DOCS_LINK } from '@/utils';
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
import {
  MultiCheckboxComponent,
  InsertClusterAttributesForm,
} from '@/components';

const ACTION_TYPE = 'type';

export function CreateActionContainer(): React.JSX.Element {
  const [currentActionType, setCurrentActionType] = useState<string>();
  const {
    actionName,
    setActionName,
    actionNote,
    setActionNote,
    selectedMonitors,
    setSelectedMonitors,
    actionData,
    setActionData,
    createNewAction,
  } = useActionState();

  const search = useSearchParams();

  useEffect(() => {
    const action = search.get(ACTION_TYPE);
    action && setCurrentActionType(action);
  }, [search]);

  function renderCurrentAction() {
    switch (currentActionType) {
      case ActionsType.INSERT_CLUSTER_ATTRIBUTES:
        return (
          <InsertClusterAttributesForm
            data={actionData}
            onChange={setActionData}
          />
        );
      default:
        return (
          <KeyvalInputWrapper>
            <KeyvalLoader />
          </KeyvalInputWrapper>
        );
    }
  }

  if (!currentActionType)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <>
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
          onSelectionChange={setSelectedMonitors}
        />
        <KeyvalInputWrapper>
          <KeyvalInput
            label={ACTIONS.ACTION_NAME}
            value={actionName}
            onChange={setActionName}
          />
        </KeyvalInputWrapper>
        {renderCurrentAction()}
        <TextareaWrapper>
          <KeyvalTextArea
            label={ACTIONS.ACTION_NOTE}
            value={actionNote}
            placeholder={ACTIONS.NOTE_PLACEHOLDER}
            onChange={(e) => setActionNote(e.target.value)}
          />
        </TextareaWrapper>
        <CreateButtonWrapper>
          <KeyvalButton onClick={createNewAction}>
            <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
              {ACTIONS.CREATE_ACTION}
            </KeyvalText>
          </KeyvalButton>
        </CreateButtonWrapper>
      </CreateActionWrapper>
    </>
  );
}
