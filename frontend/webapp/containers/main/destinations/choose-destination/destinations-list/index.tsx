import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { DestinationTypeItem } from '@/types';
import { Text } from '@/reuseable-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  align-self: stretch;
  border-radius: 16px;
  height: 100%;
  max-height: 548px;
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

const DestinationIconWrapper = styled.div`
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

const SignalsWrapper = styled.div`
  display: flex;
  gap: 4px;
`;

const SignalText = styled(Text)`
  color: rgba(249, 249, 249, 0.8);
  font-size: 10px;
  text-transform: capitalize;
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

interface DestinationsListProps {
  items: DestinationTypeItem[];
  selectedItems: DestinationTypeItem[];
  setSelectedItems: (item: DestinationTypeItem) => void;
}

const DestinationsList: React.FC<DestinationsListProps> = ({
  items,
  selectedItems,
  setSelectedItems,
}) => {
  function renderSupportedSignals(item: DestinationTypeItem) {
    const supportedSignals = item.supportedSignals;
    const signals = Object.keys(supportedSignals);
    const supportedSignalsList = signals.filter(
      (signal) => supportedSignals[signal].supported
    );

    return supportedSignalsList.map((signal, index) => (
      <>
        <SignalText>{signal}</SignalText>
        {index < supportedSignalsList.length - 1 && <SignalText>·</SignalText>}
      </>
    ));
  }
  renderSupportedSignals(items[0]);
  return (
    <Container>
      {items.map((item) => (
        <ListItem
          key={item.displayName}
          selected={selectedItems.includes(item)}
          onClick={() => setSelectedItems(item)}
        >
          <ListItemContent>
            <DestinationIconWrapper>
              <Image
                src={item.imageUrl}
                width={20}
                height={20}
                alt="destination"
              />
            </DestinationIconWrapper>
            <TextWrapper>
              <Text>{item.displayName}</Text>
              <SignalsWrapper>{renderSupportedSignals(item)}</SignalsWrapper>
            </TextWrapper>
          </ListItemContent>
          {selectedItems.includes(item) && (
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

export { DestinationsList };
