'use client';
import React, { useEffect } from 'react';
import theme from '@/styles/palette';
import { useActionState } from '@/hooks';
import { useSearchParams } from 'next/navigation';
import { ACTION, ACTIONS, ACTION_ITEM_DOCS_LINK } from '@/utils';
import {
  MultiCheckboxComponent,
  DynamicActionForm,
  ActionIcon,
} from '@/components';
import {
  KeyvalButton,
  KeyvalInput,
  KeyvalLink,
  KeyvalLoader,
  KeyvalText,
  KeyvalTextArea,
} from '@/design.system';
import {
  Container,
  CreateActionWrapper,
  CreateButtonWrapper,
  DescriptionWrapper,
  HeaderText,
  KeyvalInputWrapper,
  LoaderWrapper,
  TextareaWrapper,
} from './styled';

const ACTION_TYPE = 'type';

export function CreateActionContainer(): React.JSX.Element {
  const { actionState, onChangeActionState, upsertAction } = useActionState();
  const { actionName, actionNote, actionData, selectedMonitors, type } =
    actionState;

  const search = useSearchParams();

  useEffect(() => {
    const action = search.get(ACTION_TYPE);
    action && onChangeActionState('type', action);
  }, [search]);

  if (!type)
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );

  return (
    <Container>
      <CreateActionWrapper>
        <HeaderText>
          <ActionIcon type={type} style={{ width: 34, height: 34 }} />
          <KeyvalText size={18} weight={700}>
            {ACTIONS[type].TITLE}
          </KeyvalText>
        </HeaderText>
        <DescriptionWrapper>
          <KeyvalText size={14}>{ACTIONS[type].DESCRIPTION}</KeyvalText>
          <KeyvalLink
            value={ACTION.LINK_TO_DOCS}
            fontSize={14}
            onClick={() =>
              window.open(
                `${ACTION_ITEM_DOCS_LINK}/${type.toLowerCase()}`,
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
          type={type}
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
    </Container>
  );
}
