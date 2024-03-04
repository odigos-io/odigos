import React from 'react';
import { useActions } from '@/hooks';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { ACTIONS, OVERVIEW, ROUTES } from '@/utils';
import { EmptyList, ActionsTable } from '@/components';
import {
  KeyvalText,
  KeyvalButton,
  KeyvalLoader,
  KeyvalSearchInput,
} from '@/design.system';
import {
  ActionsContainer,
  Container,
  Content,
  Header,
  HeaderRight,
} from './styled';

export function ManagedActionsContainer() {
  const router = useRouter();
  const { isLoading, actions } = useActions();

  function handleAddAction() {
    router.push(ROUTES.CHOOSE_ACTIONS);
  }

  function handleEditAction(id: string) {
    router.push(`${ROUTES.EDIT_ACTION}?id=${id}`);
  }

  if (isLoading) return <KeyvalLoader />;

  return (
    <Container>
      {!actions?.length ? (
        <EmptyList
          title={OVERVIEW.EMPTY_ACTION}
          btnTitle={OVERVIEW.ADD_NEW_ACTION}
          btnAction={handleAddAction}
        />
      ) : (
        <ActionsContainer>
          <Header>
            <KeyvalSearchInput
              containerStyle={{ padding: '6px 8px' }}
              placeholder={ACTIONS.SEARCH_ACTION}
            />
            <HeaderRight>
              <KeyvalButton onClick={handleAddAction} style={{ height: 32 }}>
                <KeyvalText
                  size={14}
                  weight={600}
                  color={theme.text.dark_button}
                >
                  {OVERVIEW.ADD_NEW_ACTION}
                </KeyvalText>
              </KeyvalButton>
            </HeaderRight>
          </Header>
          <Content>
            <ActionsTable data={actions} onRowClick={handleEditAction} />
          </Content>
        </ActionsContainer>
      )}
    </Container>
  );
}
