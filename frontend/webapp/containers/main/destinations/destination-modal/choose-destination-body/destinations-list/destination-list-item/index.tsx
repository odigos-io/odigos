import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { DestinationTypeItem } from '@/types';
import { Text } from '@/reuseable-components';

const HoverTextWrapper = styled.div`
  visibility: hidden;
`;

const ListItem = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 0px;
  transition: background 0.3s;
  border-radius: 16px;
  cursor: pointer;
  background: rgba(249, 249, 249, 0.04);

  &:hover {
    background: rgba(249, 249, 249, 0.08);
  }
  &:last-child {
    margin-bottom: 24px;
  }

  &:hover {
    ${HoverTextWrapper} {
      visibility: visible;
    }
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
  background: linear-gradient(180deg, rgba(249, 249, 249, 0.06) 0%, rgba(249, 249, 249, 0.02) 100%);
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

const HoverText = styled(Text)`
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-transform: uppercase;
  margin-right: 16px;
`;

interface DestinationListItemProps {
  item: DestinationTypeItem;
  onSelect: (item: DestinationTypeItem) => void;
}

export const DestinationListItem: React.FC<DestinationListItemProps> = ({ item, onSelect }) => {
  const renderSupportedSignals = () => {
    const signals = Object.keys(item.supportedSignals).filter((signal) => item.supportedSignals[signal].supported);

    return signals.map((signal, index) => (
      <SignalsWrapper key={index}>
        <SignalText>{signal}</SignalText>
        {index < signals.length - 1 && <SignalText>·</SignalText>}
      </SignalsWrapper>
    ));
  };

  return (
    <ListItem data-id={`destination-${item.displayName}`} onClick={() => onSelect(item)}>
      <ListItemContent>
        <DestinationIconWrapper>
          <Image src={item.imageUrl} width={20} height={20} alt='destination' />
        </DestinationIconWrapper>
        <TextWrapper>
          <Text size={14}>{item.displayName}</Text>
          <SignalsWrapper>{renderSupportedSignals()}</SignalsWrapper>
        </TextWrapper>
      </ListItemContent>
      <HoverTextWrapper>
        <HoverText size={14}>{'Select'}</HoverText>
      </HoverTextWrapper>
    </ListItem>
  );
};
