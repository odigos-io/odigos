'use client';
import React, { useEffect, useState } from 'react';
import theme from '@/styles/palette';
import { ActionsType } from '@/types';
import { useSearchParams } from 'next/navigation';
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
  TextareaWrapper,
} from './styled';
import {
  MultiCheckboxComponent,
  InsertClusterAttributesForm,
} from '@/components';
import { ACTION, ACTION_DOCS_LINK } from '@/utils';
import { useActionState } from '@/hooks';

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

  return (
    <>
      <CreateActionWrapper>
        <DescriptionWrapper>
          <KeyvalText size={14}>
            {`The "Insert Cluster Attribute" Odigos Action can be used to add resource attributes to telemetry signals originated from the k8s cluster where the Odigos is running.`}
          </KeyvalText>
          <KeyvalLink
            value={ACTION.LINK_TO_DOCS}
            fontSize={14}
            onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
          />
        </DescriptionWrapper>
        <MultiCheckboxComponent
          title="This action monitors"
          checkboxes={selectedMonitors}
          onSelectionChange={setSelectedMonitors}
        />
        <KeyvalInputWrapper>
          <KeyvalInput
            label="Action Name"
            value={actionName}
            onChange={setActionName}
          />
        </KeyvalInputWrapper>
        {renderCurrentAction()}
        <TextareaWrapper>
          <KeyvalTextArea
            label="Note"
            value={actionNote}
            placeholder="Add a note"
            onChange={(e) => setActionNote(e.target.value)}
          />
        </TextareaWrapper>
        <CreateButtonWrapper>
          <KeyvalButton onClick={createNewAction}>
            <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
              Create Action
            </KeyvalText>
          </KeyvalButton>
        </CreateButtonWrapper>
      </CreateActionWrapper>
    </>
  );
}
