import React from 'react';
import { useActions } from '@/hooks';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { KeyvalLoader } from '@/design.system';
import { AddItemMenu, EmptyList, ManagedActionCard } from '@/components';
import { ActionsListWrapper } from '../choose-action/styled';
import { func } from 'prop-types';

export function ManagedActionsContainer() {
  const router = useRouter();
  const { isLoading, actions } = useActions();

  function handleAddAction() {
    router.push(ROUTES.CHOOSE_ACTIONS);
  }

  function handleEditAction(id: string) {
    router.push(`${ROUTES.EDIT_ACTION}?id=${id}`);
  }

  function renderManagedActionsList() {
    return actions.map((item) => {
      return (
        <div key={item.id}>
          <ManagedActionCard
            item={item}
            onClick={() => handleEditAction(item.id)}
          />
        </div>
      );
    });
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
          <ActionsListWrapper>{renderManagedActionsList()}</ActionsListWrapper>
        </>
      )}
    </>
  );
}
