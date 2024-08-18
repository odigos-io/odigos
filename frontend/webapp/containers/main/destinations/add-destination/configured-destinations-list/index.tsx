import React, { useEffect } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { ConfiguredDestination, DestinationTypeItem } from '@/types';
import { Text } from '@/reuseable-components';
import { useSelector } from 'react-redux';
import { IAppState } from '@/store';

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

const ListItem = styled.div<{}>`
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
    background: rgba(68, 74, 217, 0.24);
  }
  &:last-child {
    margin-bottom: 32px;
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
  align-items: center;
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

interface DestinationsListProps {}

const ConfiguredDestinationsList: React.FC<DestinationsListProps> = ({}) => {
  const destinations = useSelector(
    ({ app }: { app: IAppState }) => app.configuredDestinationsList
  );

  function renderSupportedSignals(item: ConfiguredDestination) {
    const supportedSignals = item.exportedSignals;
    const signals = Object.keys(supportedSignals);
    const supportedSignalsList = signals.filter(
      (signal) => supportedSignals[signal].supported
    );

    return Object.keys(supportedSignals).map(
      (signal, index) =>
        supportedSignals[signal] && (
          <SignalsWrapper key={index}>
            <Image
              src={`/icons/monitors/${signal}.svg`}
              alt="monitor"
              width={10}
              height={16}
            />

            <SignalText>{signal}</SignalText>
            {index < supportedSignalsList.length - 1 && (
              <SignalText>·</SignalText>
            )}
          </SignalsWrapper>
        )
    );
  }

  return (
    <Container>
      {destinations.map((item) => (
        <ListItem key={item.displayName}>
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
              <Text size={14}>{item.displayName}</Text>
              <SignalsWrapper>{renderSupportedSignals(item)}</SignalsWrapper>
            </TextWrapper>
          </ListItemContent>
        </ListItem>
      ))}
    </Container>
  );
};

export { ConfiguredDestinationsList };
