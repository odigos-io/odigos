import React, { useEffect, useState } from 'react';
import { useActions, useNotification, useSources } from '@/hooks';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { ACTIONS, NOTIFICATION, OVERVIEW, ROUTES } from '@/utils';
import { EmptyList, ActionsTable, ManagedSourcesTable } from '@/components';
import {
  KeyvalText,
  KeyvalButton,
  KeyvalLoader,
  KeyvalSearchInput,
} from '@/design.system';
import {
  SourcesContainer,
  Container,
  Content,
  Header,
  HeaderRight,
} from './styled';
import { ManagedSource } from '@/types';

export function ManagedSourcesContainer() {
  const [searchInput, setSearchInput] = useState('');

  const router = useRouter();
  const { show, Notification } = useNotification();
  const {
    actions,
    sortActions,
    filterActionsBySignal,
    toggleActionStatus,
    // refetch,
  } = useActions();

  const { sources, isLoading } = useSources();

  useEffect(() => {
    console.log({ sources });
  }, [sources]);

  // useEffect(() => {
  //   refetch();
  // }, []);

  function handleAddSources() {
    router.push(ROUTES.CREATE_SOURCE);
  }

  function handleEditAction(source: ManagedSource) {
    router.push(
      `${ROUTES.MANAGE_SOURCE}?name=${source?.name}&kind=${source?.kind}&namespace=${source?.namespace}`
    );
  }

  // function filterActions() {
  //   return actions.filter(
  //     ({ spec: { actionName } }) =>
  //       actionName &&
  //       actionName.toLowerCase().includes(searchInput.toLowerCase())
  //   );
  // }

  // async function onSelectStatus(ids: string[], disabled: boolean) {
  //   const res = await toggleActionStatus(ids, disabled);

  //   show({
  //     type: res ? NOTIFICATION.SUCCESS : NOTIFICATION.ERROR,
  //     message: res
  //       ? OVERVIEW.ACTION_UPDATE_SUCCESS
  //       : OVERVIEW.ACTION_UPDATE_ERROR,
  //   });
  // }

  if (isLoading) return <KeyvalLoader />;

  return (
    <>
      <Notification />
      <Container>
        {!actions?.length ? (
          <EmptyList
            title={OVERVIEW.EMPTY_SOURCE}
            btnTitle={OVERVIEW.ADD_NEW_SOURCE}
            btnAction={handleAddSources}
          />
        ) : (
          <SourcesContainer>
            <Header>
              <KeyvalSearchInput
                containerStyle={{ padding: '6px 8px' }}
                placeholder={OVERVIEW.SEARCH_SOURCE}
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
              />
              <HeaderRight>
                <KeyvalButton onClick={handleAddSources} style={{ height: 32 }}>
                  <KeyvalText
                    size={14}
                    weight={600}
                    color={theme.text.dark_button}
                  >
                    {OVERVIEW.ADD_NEW_SOURCE}
                  </KeyvalText>
                </KeyvalButton>
              </HeaderRight>
            </Header>
            <Content>
              <ManagedSourcesTable
                data={sources}
                onRowClick={handleEditAction}
                sortActions={sortActions}
                filterActionsBySignal={filterActionsBySignal}
                // toggleActionStatus={onSelectStatus}
              />
            </Content>
          </SourcesContainer>
        )}
      </Container>
    </>
  );
}
