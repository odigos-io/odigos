import React from 'react';
import { OVERVIEW } from '@/utils';
import { AddItemMenu, EmptyList } from '@/components';

export function ManagedActionsContainer() {
  return (
    <>
      {true ? (
        <EmptyList
          title={OVERVIEW.EMPTY_ACTION}
          btnTitle={OVERVIEW.ADD_NEW_ACTION}
          btnAction={() => {}}
        />
      ) : (
        <>
          <AddItemMenu
            btnLabel={OVERVIEW.ADD_NEW_ACTION}
            length={0}
            onClick={() => {}}
            lengthLabel={OVERVIEW.MENU.ACTIONS}
          />
        </>
      )}
    </>
  );
}
