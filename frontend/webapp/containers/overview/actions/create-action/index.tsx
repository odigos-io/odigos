'use client';
import { ActionsType } from '@/types';
import { useSearchParams } from 'next/navigation';
import React, { useEffect, useState } from 'react';
import { CreateActionWrapper, KeyvalInputWrapper } from './styled';
import { KeyvalInput } from '@/design.system';
import { MultiCheckboxComponent } from '@/components';

export function CreateActionContainer(): React.JSX.Element {
  const [currentAction, setCurrentAction] = useState<string>();
  const search = useSearchParams();

  useEffect(() => {
    const action = search.get('type');
    action && setCurrentAction(action);
  }, [search]);

  function renderCurrentAction() {
    switch (currentAction) {
      case ActionsType.INSERT_CLUSTER_ATTRIBUTES:
        return <>INSERT_CLUSTER_ATTRIBUTES</>;
      default:
        return null;
    }
  }

  //TODO: 1.render action name
  //TODO: 2.render action signals
  //TODO: 3.render action form

  return (
    <CreateActionWrapper>
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
        <KeyvalInput
          label="Action Name"
          value={''}
          onChange={function (value: string): void {
            throw new Error('Function not implemented.');
          }}
        />
      </KeyvalInputWrapper>
      {renderCurrentAction()}
    </CreateActionWrapper>
  );
}
