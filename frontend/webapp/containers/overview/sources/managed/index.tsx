import React, { useState } from 'react';
import { useNotification, useSources } from '@/hooks';
import theme from '@/styles/palette';
import { useRouter } from 'next/navigation';
import { OVERVIEW, ROUTES } from '@/utils';
import { EmptyList, ManagedSourcesTable } from '@/components';
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
    sources,
    isLoading,
    sortSources,
    filterSourcesByNamespace,
    instrumentedNamespaces,
  } = useSources();

  function handleAddSources() {
    router.push(ROUTES.CREATE_SOURCE);
  }

  function handleEditSource(source: ManagedSource) {
    router.push(
      `${ROUTES.MANAGE_SOURCE}?name=${source?.name}&kind=${source?.kind}&namespace=${source?.namespace}`
    );
  }

  function filterSources() {
    return sources.filter(
      (source) =>
        source.name.toLowerCase().includes(searchInput.toLowerCase()) ||
        source.namespace.toLowerCase().includes(searchInput.toLowerCase())
    );
  }

  if (isLoading) return <KeyvalLoader />;

  return (
    <>
      <Notification />
      <Container>
        {!sources?.length ? (
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
                data={searchInput ? filterSources() : sources}
                onRowClick={handleEditSource}
                sortSources={sortSources}
                filterSourcesByNamespace={filterSourcesByNamespace}
                namespaces={instrumentedNamespaces}
              />
            </Content>
          </SourcesContainer>
        )}
      </Container>
    </>
  );
}
