import React from 'react';
import styled, { useTheme } from 'styled-components';
import { type UseSourceFormDataResponse } from '@/hooks';
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
  filterNamespaces,
  filterSources,

  selectedNamespace,
  onSelectNamespace,
  selectedSources,
  onSelectSource,
  selectedFutureApps,
  onSelectFutureApps,

  selectAllForNamespace,
  onSelectAll,
}) => {
  const theme = useTheme();
  const namespaces = filterNamespaces();

  if (!namespaces.length) {
    return <NoDataFoundWrapper>{namespacesLoading ? <FadeLoader style={{ transform: 'scale(2)' }} /> : <NoDataFound title='No namespaces found' />}</NoDataFoundWrapper>;
  }

  return (
    <List $isModal={isModal}>
      {namespaces.map(([namespace, sources]) => {
        const sourcesForNamespace = selectedSources[namespace] || [];
        const futureAppsForNamespace = selectedFutureApps[namespace] || false;
        const isNamespaceLoaded = !!sourcesForNamespace.length;
        const isNamespaceSelected = selectedNamespace === namespace && !selectAllForNamespace;

        const onlySelectedSources = sourcesForNamespace.filter(({ selected }) => selected);
        const filteredSources = filterSources(namespace, { cancelSearch: true });

        const isNamespaceAllSourcesSelected = !!onlySelectedSources.length && onlySelectedSources.length === sources.length;
        const hasFilteredSources = !!filteredSources.length;

        return (
          <Group key={`namespace-${namespace}`} data-id={`namespace-${namespace}`} $selected={isNamespaceAllSourcesSelected} $isOpen={isNamespaceSelected && hasFilteredSources}>
            <NamespaceItem $selected={isNamespaceAllSourcesSelected} onClick={() => onSelectNamespace(namespace)}>
              <FlexRow>
                <Checkbox value={isNamespaceAllSourcesSelected} onChange={(bool) => onSelectAll(bool, namespace)} />
                <Text>{namespace}</Text>
              </FlexRow>

              <FlexRow>
                <Toggle title='Include Future Sources' initialValue={futureAppsForNamespace} onChange={(bool) => onSelectFutureApps(bool, namespace)} />
                <Divider orientation='vertical' length='12px' margin='0' />
                <SelectionCount size={10} color={theme.text.grey}>
                  {isNamespaceLoaded ? `${onlySelectedSources.length}/${sources.length}` : null}
                </SelectionCount>
                <ExtendIcon extend={isNamespaceSelected} />
              </FlexRow>
            </NamespaceItem>

            {isNamespaceSelected &&
              (hasFilteredSources ? (
                <RelativeWrapper>
                  <AbsoluteWrapper>
                    <Divider orientation='vertical' length={`${filteredSources.length * 36 - 12}px`} />
                  </AbsoluteWrapper>

                  {filteredSources.map((source) => {
                    const isSourceSelected = !!onlySelectedSources.find(({ name }) => name === source.name);

                    return (
                      <SourceItem key={`source-${source.name}`} data-id={`source-${source.name}`} $selected={isSourceSelected} onClick={() => onSelectSource(source)}>
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
