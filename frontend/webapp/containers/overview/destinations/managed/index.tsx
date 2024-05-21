'use client';
import React, { useEffect, useState } from 'react';
import {
  KeyvalButton,
  KeyvalLoader,
  KeyvalSearchInput,
  KeyvalText,
} from '@/design.system';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { useDestinations, useNotification } from '@/hooks';
import { EmptyList, ManagedDestinationsTable } from '@/components';
import {
  Header,
  Content,
  Container,
  HeaderRight,
  DestinationsContainer,
} from './styled';

export function DestinationContainer() {
  const [searchInput, setSearchInput] = useState('');
  const {
    destinationList,
    destinationLoading,
    sortDestinations,
    refetchDestinations,
    filterDestinationsBySignal,
  } = useDestinations();

  const { show, Notification } = useNotification();

  const router = useRouter();

  useEffect(() => {
    refetchDestinations();
  }, []);

  function handleAddDestination() {
    router.push(ROUTES.CREATE_DESTINATION);
  }

  function filterDestinations() {
    return destinationList.filter((destination) =>
      destination.name.toLowerCase().includes(searchInput.toLowerCase())
    );
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <Container>
      {!destinationList?.length ? (
        <EmptyList
          title={OVERVIEW.EMPTY_DESTINATION}
          btnTitle={OVERVIEW.ADD_NEW_DESTINATION}
          btnAction={handleAddDestination}
        />
      ) : (
        <DestinationsContainer>
          <Header>
            <KeyvalSearchInput
              containerStyle={{ padding: '6px 8px' }}
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
            />
            <HeaderRight>
              <KeyvalButton
                onClick={handleAddDestination}
                style={{ height: 32 }}
              >
                <KeyvalText
                  size={14}
                  weight={600}
                  color={theme.text.dark_button}
                >
                  {OVERVIEW.ADD_NEW_DESTINATION}
                </KeyvalText>
              </KeyvalButton>
            </HeaderRight>
          </Header>
          <Content>
            <ManagedDestinationsTable
              sortDestinations={sortDestinations}
              filterDestinationsBySignal={filterDestinationsBySignal}
              data={searchInput ? filterDestinations() : destinationList}
              onRowClick={({ id }) =>
                router.push(`${ROUTES.UPDATE_DESTINATION}${id}`)
              }
            />
          </Content>
        </DestinationsContainer>
      )}
      <Notification />
    </Container>
  );
}
