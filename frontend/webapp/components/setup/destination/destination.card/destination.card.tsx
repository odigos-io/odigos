import React, { useMemo } from 'react';
import { KeyvalCard, KeyvalImage, KeyvalText } from '@/design.system';
import { TapList } from '../tap.list/tap.list';
import { MONITORING_OPTIONS } from '../../../../utils/constants/monitors';
import {
  ApplicationNameWrapper,
  DestinationCardContentWrapper,
  DestinationCardWrapper,
} from './destination.card.styled';

const TEXT_STYLE: React.CSSProperties = {
  overflowWrap: 'break-word',
  textAlign: 'center',
};
const LOGO_STYLE: React.CSSProperties = {
  padding: 4,
  backgroundColor: '#fff',
  width: '6vh',
  height: '6vh',
};
const TAP_STYLE: React.CSSProperties = { padding: '4px 8px', gap: 4 };

type Destination = {
  supported_signals: {
    [key: string]: {
      supported: boolean;
    };
  };
  image_url: string;
  display_name: string;
};

type DestinationCardProps = {
  item: Destination;
  onClick: () => void;
};

export function DestinationCard({
  item: { supported_signals, image_url, display_name },
  onClick,
}: DestinationCardProps) {
  const monitors = useMemo(() => {
    return Object.entries(supported_signals).reduce((acc, [key, value]) => {
      if (value.supported) {
        const monitor = MONITORING_OPTIONS.find(
          (option) => option.title.toLowerCase() === key
        );
        if (monitor) {
          return [...acc, { ...monitor, tapped: true }];
        }
      }
      return acc;
    }, []);
  }, [JSON.stringify(supported_signals)]);

  return (
    <KeyvalCard>
      <DestinationCardWrapper data-cy={'choose-destination-'+ display_name} onClick={onClick}>
        <KeyvalImage
          src={image_url}
          width={56}
          height={56}
          style={LOGO_STYLE}
        />
        <DestinationCardContentWrapper>
          <ApplicationNameWrapper>
            <KeyvalText size={20} weight={700} style={TEXT_STYLE}>
              {display_name}
            </KeyvalText>
          </ApplicationNameWrapper>
          <TapList gap={4} list={monitors} tapStyle={TAP_STYLE} />
        </DestinationCardContentWrapper>
      </DestinationCardWrapper>
    </KeyvalCard>
  );
}
