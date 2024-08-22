import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { K8sActualSource } from '@/types';
import { NoDataFound, Text } from '@/reuseable-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  background: ${({ theme }) => theme.colors.primary};
  height: 100%;
  max-height: calc(100vh - 360px);
  overflow-y: auto;
`;

const ListItem = styled.div<{ selected: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 0px;
  transition: background 0.3s;
  border-radius: 16px;

  cursor: pointer;
  background: ${({ selected }) =>
    selected ? 'rgba(68, 74, 217, 0.24)' : 'rgba(249, 249, 249, 0.04)'};

  &:hover {
    background: rgba(68, 74, 217, 0.24);
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
  background: linear-gradient(
    180deg,
    rgba(249, 249, 249, 0.06) 0%,
    rgba(249, 249, 249, 0.02) 100%
  );
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

const NoDataFoundWrapper = styled(Container)`
  margin-top: 80px;
`;

interface SourcesListProps {
  items: K8sActualSource[];
  selectedItems: K8sActualSource[];
  setSelectedItems: (item: K8sActualSource) => void;
}

const SourcesList: React.FC<SourcesListProps> = ({
  items,
  selectedItems,
  setSelectedItems,
}) => {
  function isItemSelected(item: K8sActualSource) {
    const selected = selectedItems.find(
      (selectedItem) => selectedItem.name === item.name
    );
    return !!selected;
  }

  if (!items.length) {
    return (
      <NoDataFoundWrapper>
        <NoDataFound title="No sources found" />
      </NoDataFoundWrapper>
    );
  }

  return (
    <Container>
      {items.map((item) => (
        <ListItem
          key={item.name}
          selected={isItemSelected(item)}
          onClick={() => setSelectedItems(item)}
        >
          <ListItemContent>
            <SourceIconWrapper>
              <Image
                src={'/icons/common/folder.svg'}
                width={20}
                height={20}
                alt="source"
              />
            </SourceIconWrapper>
            <TextWrapper>
              <Text>{item.name}</Text>
              <Text opacity={0.8} size={10}>
                {item.numberOfInstances} running instances · {item.kind}
              </Text>
            </TextWrapper>
          </ListItemContent>
          {isItemSelected(item) && (
            <SelectedTextWrapper>
              <Text size={12} family="secondary">
                SELECTED
              </Text>
            </SelectedTextWrapper>
          )}
        </ListItem>
      ))}
    </Container>
  );
};

export { SourcesList };
