import React, { useState } from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { DeleteWarning } from '@/components';
import { IAppState, useAppStore } from '@/store';
import { OVERVIEW_ENTITY_TYPES, type ConfiguredDestination } from '@/types';
import { Button, DataCardFields, Divider, ExtendIcon, Text } from '@/reuseable-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  margin-top: 24px;
  max-height: calc(100vh - 400px);
  height: 100%;
  overflow-x: hidden;
  overflow-y: scroll;
`;

const ListItem = styled.div`
  width: 100%;
  border-radius: 16px;
  background: ${({ theme }) => theme.colors.translucent_bg};
`;

const ListItemBody = styled.div`
  width: 100%;
  padding: 16px;
`;

const ListItemHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 0px;
`;

const ListItemContent = styled.div`
  display: flex;
  gap: 12px;
  margin-left: 16px;
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

const IconsContainer = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  margin-right: 16px;
`;

const IconButton = styled(Button)<{ $expand?: boolean }>`
  transition: background 0.3s ease 0s, transform 0.3s ease 0s;
  transform: ${({ $expand }) => ($expand ? 'rotate(-180deg)' : 'rotate(0deg)')};
`;

const ConfiguredDestinationsListItem: React.FC<{ item: ConfiguredDestination; isLastItem: boolean }> = ({ item, isLastItem }) => {
  const [expand, setExpand] = useState(false);
  const [deleteWarning, setDeleteWarning] = useState(false);
  const { removeConfiguredDestination } = useAppStore((state) => state);

  function renderSupportedSignals(item: ConfiguredDestination) {
    const supportedSignals = item.exportedSignals;
    const signals = Object.keys(supportedSignals);
    const supportedSignalsList = signals.filter((signal) => supportedSignals[signal].supported);

    return Object.keys(supportedSignals).map(
      (signal, index) =>
        supportedSignals[signal] && (
          <SignalsWrapper key={index}>
            <Image src={`/icons/monitors/${signal}.svg`} alt='monitor' width={10} height={16} />

            <SignalText>{signal}</SignalText>
            {index < supportedSignalsList.length - 1 && <SignalText>·</SignalText>}
          </SignalsWrapper>
        ),
    );
  }

  return (
    <>
      <ListItem>
        <ListItemHeader style={{ paddingBottom: expand ? 0 : 16 }}>
          <ListItemContent>
            <DestinationIconWrapper>
              <Image src={item.imageUrl} alt='destination' width={20} height={20} />
            </DestinationIconWrapper>
            <TextWrapper>
              <Text size={14}>{item.displayName}</Text>
              <SignalsWrapper>{renderSupportedSignals(item)}</SignalsWrapper>
            </TextWrapper>
          </ListItemContent>

          <IconsContainer>
            <IconButton variant='tertiary' onClick={() => setDeleteWarning(true)}>
              <Image src='/icons/common/trash.svg' alt='delete' width={16} height={16} />
            </IconButton>
            <Divider orientation='vertical' length='16px' />
            <IconButton variant='tertiary' onClick={() => setExpand(!expand)}>
              <ExtendIcon extend={expand} />
            </IconButton>
          </IconsContainer>
        </ListItemHeader>

        {expand && (
          <ListItemBody>
            <Divider margin='0 0 16px 0' length='calc(100% - 32px)' />
            <DataCardFields data={item.destinationTypeDetails} />
          </ListItemBody>
        )}
      </ListItem>

      <DeleteWarning
        isOpen={deleteWarning}
        name={item.displayName || item.type}
        type={OVERVIEW_ENTITY_TYPES.DESTINATION}
        isLastItem={isLastItem}
        onApprove={() => removeConfiguredDestination(item)}
        onDeny={() => setDeleteWarning(false)}
      />
    </>
  );
};

export const ConfiguredDestinationsList: React.FC<{ data: IAppState['configuredDestinations'] }> = ({ data }) => {
  return (
    <Container>
      {data.map(({ stored }) => (
        <ConfiguredDestinationsListItem key={stored.displayName} item={stored} isLastItem={data.length === 1} />
      ))}
    </Container>
  );
};
