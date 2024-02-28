import React, { useEffect } from 'react';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { AddItemMenu, EmptyList } from '@/components';
import { useActions } from '@/hooks';
import { KeyvalLoader } from '@/design.system';

export function ManagedActionsContainer() {
  const router = useRouter();
  const { isLoading, actions } = useActions();

  useEffect(() => {
    console.log({ actions, isLoading });
  }, [actions]);

  function handleAddAction() {
    router.push(ROUTES.CHOOSE_ACTIONS);
  }

  if (isLoading) return <KeyvalLoader />;

  return (
    <>
      {!actions?.length ? (
        <EmptyList
          title={OVERVIEW.EMPTY_ACTION}
          btnTitle={OVERVIEW.ADD_NEW_ACTION}
          btnAction={handleAddAction}
        />
      ) : (
        <>
          <AddItemMenu
            btnLabel={OVERVIEW.ADD_NEW_ACTION}
            length={actions.length}
            onClick={handleAddAction}
            lengthLabel={OVERVIEW.MENU.ACTIONS}
          />
        </>
      )}
    </>
  );
}
