import React from 'react';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { AddItemMenu, EmptyList } from '@/components';

export function ManagedActionsContainer() {
  const router = useRouter();

  function handleAddAction() {
    router.push(ROUTES.CHOOSE_ACTIONS);
  }
  return (
    <>
      {true ? (
        <EmptyList
          title={OVERVIEW.EMPTY_ACTION}
          btnTitle={OVERVIEW.ADD_NEW_ACTION}
          btnAction={handleAddAction}
        />
      ) : (
        <>
          <AddItemMenu
            btnLabel={OVERVIEW.ADD_NEW_ACTION}
            length={0}
            onClick={handleAddAction}
            lengthLabel={OVERVIEW.MENU.ACTIONS}
          />
        </>
      )}
    </>
  );
}
