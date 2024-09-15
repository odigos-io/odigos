import React, { useEffect, useRef, useState } from 'react';
import theme from '@/styles/palette';
import { useSources } from '@/hooks';
import { ManagedSource } from '@/types';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter, useSearchParams } from 'next/navigation';
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

const POLL_DATA = 'poll';
const POLL_INTERVAL = 2000; // Interval in milliseconds between polls
const MAX_ATTEMPTS = 5; // Maximum number of polling attempts

export function ManagedSourcesContainer() {
  const [searchInput, setSearchInput] = useState('');
  const [pollingAttempts, setPollingAttempts] = useState(0);

  const router = useRouter();
  const useSearch = useSearchParams();
  const intervalId = useRef<NodeJS.Timer>();

  const {
    sources,
    isLoading,
    sortSources,
    refetchSources,
    filterSourcesByKind,
    deleteSourcesHandler,
    filterSourcesByLanguage,
    instrumentedNamespaces,
    filterSourcesByNamespace,
    filterByConditionStatus,
    filterByConditionMessage,
  } = useSources();

  useEffect(() => {
    const pullData = useSearch.get(POLL_DATA);
    if (pullData) {
      intervalId.current = setInterval(() => {
        Promise.all([refetchSources()])
          .then(() => {})
          .catch(console.error);

        setPollingAttempts((prev) => prev + 1);
      }, POLL_INTERVAL);

      return () => clearInterval(intervalId.current);
    }
  }, [refetchSources, pollingAttempts, useSearch]);

  useEffect(() => {
    if (pollingAttempts >= MAX_ATTEMPTS) {
      clearInterval(intervalId.current);
      return;
    }
  }, [pollingAttempts]);

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
                namespaces={instrumentedNamespaces}
                filterSourcesByKind={filterSourcesByKind}
                filterByConditionStatus={filterByConditionStatus}
                deleteSourcesHandler={deleteSourcesHandler}
                data={searchInput ? filterSources() : sources}
                filterSourcesByLanguage={filterSourcesByLanguage}
                filterSourcesByNamespace={filterSourcesByNamespace}
                filterByConditionMessage={filterByConditionMessage}
              />
            </Content>
          </SourcesContainer>
        )}
      </Container>
    </>
  );
}
