import React from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { UseSourceFormDataResponse } from '@/hooks';
import { Checkbox, Divider, NoDataFound, Text, Toggle } from '@/reuseable-components';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

const List = styled.div<{ isModal: boolean }>`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  max-height: ${({ isModal }) => (isModal ? 'calc(100vh - 548px)' : 'calc(100vh - 360px)')};
  height: fit-content;
  padding-bottom: ${({ isModal }) => (isModal ? '48px' : '0')};
  overflow-y: scroll;
`;

const Group = styled.div<{ isSelected: boolean; isOpen: boolean }>`
  width: 100%;
  padding-bottom: ${({ isOpen }) => (isOpen ? '18px' : '0')};
  border-radius: 16px;
  background-color: ${({ isSelected }) => (isSelected ? 'rgba(68, 74, 217, 0.24)' : 'rgba(249, 249, 249, 0.04)')};
`;

const NamespaceItem = styled.div<{ isSelected: boolean }>`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin: 0;
  padding: 18px;
  border-radius: 16px;
  cursor: pointer;
  &:hover {
    background-color: ${({ isSelected }) => (isSelected ? 'rgba(68, 74, 217, 0.40)' : 'rgba(249, 249, 249, 0.08)')};
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

  selectedNamespace,
  onSelectNamespace,
  availableSources,
  selectedSources,
  onSelectSource,
  selectedFutureApps,
  onSelectFutureApps,

  searchText,
  setSearchText,
  selectAll,
  onSelectAll,
  showSelectedOnly,
  setShowSelectedOnly,

  filterSources,
}) => {
  const namespaces = Object.entries(availableSources);

  if (!namespaces.length) {
    return (
      <NoDataFoundWrapper>
        <NoDataFound title='No namespaces found' />
      </NoDataFoundWrapper>
    );
  }

  return (
    <List isModal={isModal}>
      {namespaces.map(([namespace, sources]) => {
        const namespaceLoaded = !!selectedSources[namespace];

        const selected = selectedSources[namespace] || [];
        const futureApps = selectedFutureApps[namespace] || false;

        const namespacePassesFilters = (!searchText || namespace.toLowerCase().includes(searchText)) && (!showSelectedOnly || !!selected.length);
        if (!namespacePassesFilters) return null;

        const isNamespaceSelected = selectedNamespace === namespace;
        const isNamespaceCanSelect = namespaceLoaded && !!selected.length;
        const isNamespaceAllSourcesSelected = isNamespaceCanSelect && selected.length === sources.length;

        const filtered = filterSources(namespace, { cancelSearch: true });
        const hasFilteredSources = !!filtered.length;

        return (
          <Group key={`namespace-${namespace}`} isSelected={isNamespaceAllSourcesSelected} isOpen={isNamespaceSelected && hasFilteredSources}>
            <NamespaceItem isSelected={isNamespaceAllSourcesSelected} onClick={() => onSelectNamespace(namespace)}>
              <FlexRow>
                <Checkbox disabled={!isNamespaceCanSelect} initialValue={isNamespaceAllSourcesSelected} onChange={(bool) => onSelectAll(bool, namespace)} />
                <Text>{namespace}</Text>
              </FlexRow>

              <FlexRow>
                <Toggle title='Future select' initialValue={futureApps} onChange={(bool) => onSelectFutureApps(bool, namespace)} />
                <Divider orientation='vertical' length='12px' margin='0' />
                <SelectionCount size={10} color={theme.text.grey}>
                  {namespaceLoaded ? `${selected.length}/${sources.length}` : null}
                </SelectionCount>
                <ArrowIcon src='/icons/common/extend-arrow.svg' alt='open-dropdown' width={14} height={14} className={isNamespaceSelected ? 'open' : 'close'} />
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
                      <SourceItem key={`source-${source.name}`} isSelected={isSourceSelected} onClick={() => onSelectSource(source)}>
                        <FlexRow>
                          <Checkbox initialValue={isSourceSelected} onChange={() => onSelectSource(source, namespace)} />
                          <Text>{source.name}</Text>
                          <Text opacity={0.8} size={10}>
                            {source.numberOfInstances} running instances · {source.kind}
                          </Text>
                        </FlexRow>
                      </SourceItem>
                    );
                  })}
                </RelativeWrapper>
              ) : (
                <NoDataFoundWrapper>
                  <NoDataFound title='No sources found' />
                </NoDataFoundWrapper>
              ))}
          </Group>
        );
      })}
    </List>
  );
};
