import React, { useState } from 'react';
import theme from '@/styles/palette';
import { useSources } from '@/hooks';
import { ManagedSource } from '@/types';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { EmptyList, ManagedSourcesTable } from '@/components';
import {
  KeyvalText,
  KeyvalLoader,
  KeyvalButton,
  KeyvalSearchInput,
} from '@/design.system';
import {
  Header,
  Content,
  Container,
  HeaderRight,
  SourcesContainer,
} from './styled';

export function ManagedSourcesContainer() {
  const [searchInput, setSearchInput] = useState('');

  const router = useRouter();

  const {
    sources,
    isLoading,
    sortSources,
    filterSourcesByKind,
    deleteSourcesHandler,
    instrumentedNamespaces,
    filterSourcesByNamespace,
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
                sortSources={sortSources}
                onRowClick={handleEditSource}
                deleteSourcesHandler={deleteSourcesHandler}
                namespaces={instrumentedNamespaces}
                filterSourcesByKind={filterSourcesByKind}
                data={searchInput ? filterSources() : sources}
                filterSourcesByNamespace={filterSourcesByNamespace}
              />
            </Content>
          </SourcesContainer>
        )}
      </Container>
    </>
  );
}
