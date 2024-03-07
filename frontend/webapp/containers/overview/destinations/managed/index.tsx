'use client';
import React, { useState } from 'react';
import {
  KeyvalButton,
  KeyvalLoader,
  KeyvalSearchInput,
  KeyvalText,
} from '@/design.system';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { EmptyList, ManagedDestinationsTable } from '@/components';
import { useRouter } from 'next/navigation';
import { useDestinations, useNotification } from '@/hooks';
import {
  Container,
  Content,
  DestinationsContainer,
  Header,
  HeaderRight,
} from './styled';
import theme from '@/styles/palette';
import { func } from 'prop-types';

export function DestinationContainer() {
  const [searchInput, setSearchInput] = useState('');
  const { destinationList, destinationLoading } = useDestinations();

  const { show, Notification } = useNotification();

  const router = useRouter();

  function handleAddDestination() {
    router.push(ROUTES.CREATE_DESTINATION);
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <Container>
      {!destinationList?.length ? (
        <EmptyList
          title={OVERVIEW.EMPTY_ACTION}
          btnTitle={OVERVIEW.ADD_NEW_ACTION}
          btnAction={handleAddDestination}
        />
      ) : (
        <DestinationsContainer>
          <Header>
            <KeyvalSearchInput
              containerStyle={{ padding: '6px 8px' }}
              //   placeholder={ACTIONS.SEARCH_ACTION}
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
              data={destinationList}
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
