import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { UseSourceFormDataResponse } from '@/hooks';
import { Checkbox, Divider, ExtendIcon, FadeLoader, NoDataFound, Text, Toggle } from '@/reuseable-components';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

const List = styled.div<{ $isModal: Props['isModal'] }>`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-height: ${({ $isModal }) => ($isModal ? 'calc(100vh - 510px)' : 'calc(100vh - 310px)')};
  height: fit-content;
  overflow-y: scroll;
`;

const Group = styled.div<{ $selected: boolean; $isOpen: boolean }>`
  width: 100%;
  padding-bottom: ${({ $isOpen }) => ($isOpen ? '18px' : '0')};
  border-radius: 16px;
  background-color: ${({ $selected }) => ($selected ? 'rgba(68, 74, 217, 0.24)' : 'rgba(249, 249, 249, 0.04)')};
`;

const NamespaceItem = styled.div<{ $selected: boolean }>`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 0;
  padding: 18px;
  border-radius: 16px;
  cursor: pointer;
  &:hover {
    background-color: ${({ $selected }) => ($selected ? 'rgba(68, 74, 217, 0.40)' : 'rgba(249, 249, 249, 0.08)')};
    transition: background-color 0.3s;
  }
`;

const SourceItem = styled(NamespaceItem)`
  width: calc(100% - 50px);
  margin-left: auto;
  padding: 8px;
`;

const FlexRow = styled.div`
  display: flex;
  align-items: center;
  gap: 12px;
`;

const RelativeWrapper = styled.div`
  position: relative;
`;

const AbsoluteWrapper = styled.div`
  position: absolute;
  top: 6px;
  left: 18px;
`;

const SelectionCount = styled(Text)`
  width: 18px;
`;

const ArrowIcon = styled(Image)`
  &.open {
    transform: rotate(180deg);
  }
  &.close {
    transform: rotate(0deg);
  }
  transition: transform 0.3s;
`;

const NoDataFoundWrapper = styled.div`
  margin: 50px 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  height: 100%;
  max-height: calc(100vh - 360px);
  overflow-y: auto;
`;

export const SourcesList: React.FC<Props> = ({
  isModal = false,
  namespacesLoading,

  selectedNamespace,
  onSelectNamespace,
  availableSources,
  selectedSources,
  onSelectSource,
  selectedFutureApps,
  onSelectFutureApps,

  searchText,
  selectAllForNamespace,
  onSelectAll,

  filterSources,
}) => {
  const namespaces = Object.entries(availableSources);

  if (!namespaces.length) {
    return <NoDataFoundWrapper>{namespacesLoading ? <FadeLoader style={{ transform: 'scale(2)' }} /> : <NoDataFound title='No namespaces found' />}</NoDataFoundWrapper>;
  }

  return (
    <List $isModal={isModal}>
      {namespaces.map(([namespace, sources]) => {
        const namespaceLoaded = !!selectedSources[namespace];

        const available = availableSources[namespace] || [];
        const selected = selectedSources[namespace] || [];
        const futureApps = selectedFutureApps[namespace] || false;

        const namespacePassesFilters = !searchText || namespace.toLowerCase().includes(searchText);
        if (!namespacePassesFilters) return null;

        const isNamespaceSelected = selectedNamespace === namespace && !selectAllForNamespace;
        const isNamespaceCanSelect = namespaceLoaded && !!available.length;
        const isNamespaceAllSourcesSelected = isNamespaceCanSelect && selected.length === sources.length;

        const filtered = filterSources(namespace, { cancelSearch: true });
        const hasFilteredSources = !!filtered.length;

        return (
          <Group data-id={`namespace-${namespace}`} key={`namespace-${namespace}`} $selected={isNamespaceAllSourcesSelected} $isOpen={isNamespaceSelected && hasFilteredSources}>
            <NamespaceItem $selected={isNamespaceAllSourcesSelected} onClick={() => onSelectNamespace(namespace)}>
              <FlexRow>
                <Checkbox disabled={namespaceLoaded && !isNamespaceCanSelect} value={isNamespaceAllSourcesSelected} onChange={(bool) => onSelectAll(bool, namespace)} />
                <Text>{namespace}</Text>
              </FlexRow>

              <FlexRow>
                <Toggle title='Include Future Sources' initialValue={futureApps} onChange={(bool) => onSelectFutureApps(bool, namespace)} />
                <Divider orientation='vertical' length='12px' margin='0' />
                <SelectionCount size={10} color={theme.text.grey}>
                  {namespaceLoaded ? `${selected.length}/${sources.length}` : null}
                </SelectionCount>
                <ExtendIcon extend={isNamespaceSelected} />
              </FlexRow>
            </NamespaceItem>

            {isNamespaceSelected &&
              (hasFilteredSources ? (
                <RelativeWrapper>
                  <AbsoluteWrapper>
                    <Divider orientation='vertical' length={`${filtered.length * 36 - 12}px`} />
                  </AbsoluteWrapper>

                  {filtered.map((source) => {
                    const isSourceSelected = !!selected.find(({ name }) => name === source.name);

                    return (
                      <SourceItem key={`source-${source.name}`} $selected={isSourceSelected} onClick={() => onSelectSource(source)}>
                        <FlexRow>
                          <Checkbox value={isSourceSelected} onChange={() => onSelectSource(source, namespace)} />
                          <Text>{source.name}</Text>
                          <Text opacity={0.8} size={10}>
                            {source.numberOfInstances} running instance{source.numberOfInstances !== 1 && 's'} · {source.kind}
                          </Text>
                        </FlexRow>
                      </SourceItem>
                    );
                  })}
                </RelativeWrapper>
              ) : (
                <NoDataFoundWrapper>
                  <NoDataFound title='No sources available in this namespace' subTitle='Try searching again or select another namespace.' />
                </NoDataFoundWrapper>
              ))}
          </Group>
        );
      })}
    </List>
  );
};
