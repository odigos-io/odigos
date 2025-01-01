import React from 'react';
import { FolderIcon } from '@/assets';
import styled from 'styled-components';
import { type UseSourceFormDataResponse } from '@/hooks';
import { Checkbox, FadeLoader, NoDataFound, Text } from '@/reuseable-components';

interface Props extends UseSourceFormDataResponse {
  isModal?: boolean;
}

const SourcesListWrapper = styled.div<{ $isModal: Props['isModal'] }>`
  display: flex;
  align-items: center;
  flex-direction: column;
  gap: 12px;
  height: fit-content;
  padding-bottom: ${({ $isModal }) => ($isModal ? '48px' : '0')};
  overflow-y: scroll;
`;

const ListItem = styled.div<{ $selected: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 0px;
  transition: background 0.3s;
  border-radius: 16px;
  cursor: pointer;
  background: ${({ $selected }) => ($selected ? 'rgba(68, 74, 217, 0.24)' : 'rgba(249, 249, 249, 0.04)')};
  &:hover {
    background: ${({ $selected }) => ($selected ? 'rgba(68, 74, 217, 0.40)' : 'rgba(249, 249, 249, 0.08)')};
  }
`;

const ListItemContent = styled.div`
  margin-left: 16px;
  display: flex;
  gap: 12px;
`;

const SourceIconWrapper = styled.div`
  display: flex;
  width: 36px;
  height: 36px;
  justify-content: center;
  align-items: center;
  gap: 8px;
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(249, 249, 249, 0.06) 0%, rgba(249, 249, 249, 0.02) 100%);
`;

const TextWrapper = styled.div`
  display: flex;
  flex-direction: column;
  height: 36px;
  justify-content: space-between;
`;

const SelectedTextWrapper = styled.div`
  margin-right: 24px;
`;

const NoDataFoundWrapper = styled.div`
  margin-top: 80px;
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
  selectedSources,
  onSelectSource,

  filterSources,
}) => {
  const sources = selectedSources[selectedNamespace] || [];

  if (!sources.length) {
    return <NoDataFoundWrapper>{namespacesLoading ? <FadeLoader style={{ transform: 'scale(2)' }} /> : <NoDataFound title='No sources found' />}</NoDataFoundWrapper>;
  }

  return (
    <SourcesListWrapper $isModal={isModal}>
      {filterSources().map((source) => {
        const isSelected = selectedSources[selectedNamespace].find(({ name }) => name === source.name)?.selected || false;

        return (
          <ListItem key={`source-${source.name}`} $selected={isSelected} onClick={() => onSelectSource(source)}>
            <ListItemContent>
              <SourceIconWrapper>
                <FolderIcon size={20} />
              </SourceIconWrapper>

              <TextWrapper>
                <Text>{source.name}</Text>
                <Text opacity={0.8} size={10}>
                  {source.numberOfInstances} running instance{source.numberOfInstances !== 1 && 's'} · {source.kind}
                </Text>
              </TextWrapper>
            </ListItemContent>

            {isSelected && (
              <SelectedTextWrapper>
                <Checkbox value={true} allowPropagation />
              </SelectedTextWrapper>
            )}
          </ListItem>
        );
      })}
    </SourcesListWrapper>
  );
};
