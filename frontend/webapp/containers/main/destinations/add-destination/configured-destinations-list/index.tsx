import React from 'react';
import Image from 'next/image';
import styled from 'styled-components';
import { ConfiguredFields } from '@/components';
import { ConfiguredDestination } from '@/types';
import { Divider, Text } from '@/reuseable-components';

const Container = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 12px;
  margin-top: 24px;
  align-self: stretch;
  height: 100%;
  max-height: 548px;
  overflow-y: auto;
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

const ExpandIconContainer = styled.div`
  display: flex;
  justify-content: center;
  align-items: center;
  margin-right: 16px;
`;

const IconBorder = styled.div`
  height: 16px;
  width: 1px;
  margin-right: 12px;
  background: ${({ theme }) => theme.colors.border};
`;

const ExpandIconWrapper = styled.div<{ $expand?: boolean }>`
  display: flex;
  width: 36px;
  height: 36px;
  cursor: pointer;
  justify-content: center;
  align-items: center;
  border-radius: 100%;
  transition: background 0.3s ease 0s, transform 0.3s ease 0s;
  transform: ${({ $expand }) => ($expand ? 'rotate(180deg)' : 'rotate(0deg)')};
  &:hover {
    background: ${({ theme }) => theme.colors.translucent_bg};
  }
`;

interface DestinationsListProps {
  data: ConfiguredDestination[];
}

function ConfiguredDestinationsListItem({ item }: { item: ConfiguredDestination }) {
  const [expand, setExpand] = React.useState(false);

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
    <ListItem>
      <ListItemHeader style={{ paddingBottom: expand ? 0 : 16 }}>
        <ListItemContent>
          <DestinationIconWrapper>
            <Image src={item.imageUrl} width={20} height={20} alt='destination' />
          </DestinationIconWrapper>
          <TextWrapper>
            <Text size={14}>{item.displayName}</Text>
            <SignalsWrapper>{renderSupportedSignals(item)}</SignalsWrapper>
          </TextWrapper>
        </ListItemContent>

        <ExpandIconContainer>
          <IconBorder />
          <ExpandIconWrapper $expand={expand} onClick={() => setExpand(!expand)}>
            <Image src={'/icons/common/extend-arrow.svg'} width={16} height={16} alt='destination' />
          </ExpandIconWrapper>
        </ExpandIconContainer>
      </ListItemHeader>

      {expand && (
        <ListItemBody>
          <Divider margin='0 0 16px 0' />
          <ConfiguredFields details={item.destinationTypeDetails} />
        </ListItemBody>
      )}
    </ListItem>
  );
}

const ConfiguredDestinationsList: React.FC<DestinationsListProps> = ({ data }) => {
  return (
    <Container>
      {data.map((item) => (
        <ConfiguredDestinationsListItem key={item.displayName} item={item} />
      ))}
    </Container>
  );
};

export { ConfiguredDestinationsList };
