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
} from '@/design.system';
import {
  CreateActionWrapper,
  CreateButtonWrapper,
  DescriptionWrapper,
  KeyvalInputWrapper,
  LinkWrapper,
} from './styled';
import {
  MultiCheckboxComponent,
  InsertClusterAttributesForm,
} from '@/components';
import { ACTION, ACTION_DOCS_LINK } from '@/utils';

const ACTION_TYPE = 'type';

export function CreateActionContainer(): React.JSX.Element {
  const [currentAction, setCurrentAction] = useState<string>();

  const search = useSearchParams();

  useEffect(() => {
    const action = search.get(ACTION_TYPE);
    action && setCurrentAction(action);
  }, [search]);

  function renderCurrentAction() {
    switch (currentAction) {
      case ActionsType.INSERT_CLUSTER_ATTRIBUTES:
        return <InsertClusterAttributesForm onChange={() => {}} />;
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
            value={ACTION.LEARN_MORE}
            fontSize={14}
            onClick={() => window.open(ACTION_DOCS_LINK, '_blank')}
          />
        </DescriptionWrapper>
        <MultiCheckboxComponent
          title="This action monitors"
          checkboxes={[
            { id: '1', label: 'Logs', checked: false },
            { id: '2', label: 'Metrics', checked: false },
            { id: '3', label: 'Traces', checked: false },
          ]}
          onSelectionChange={() => {}}
        />
        <KeyvalInputWrapper>
          <KeyvalInput label="Action Name" value={''} onChange={() => {}} />
        </KeyvalInputWrapper>
        {renderCurrentAction()}
        <CreateButtonWrapper>
          <KeyvalButton disabled>
            <KeyvalText weight={600} color={theme.text.dark_button} size={14}>
              Create Action
            </KeyvalText>
          </KeyvalButton>
        </CreateButtonWrapper>
      </CreateActionWrapper>
    </>
  );
}
