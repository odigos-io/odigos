import React from 'react';
import { KeyvalText } from '@/design.system';
import { HeaderTitleWrapper, SetupHeaderWrapper } from './styled';
import { ConnectionsIcons } from '../../connection/connections_icons/connections_icons';

export function ConnectDestinationHeader({ icon, name }) {
  return (
    <SetupHeaderWrapper>
      <HeaderTitleWrapper>
        <ConnectionsIcons icon={icon} />
        <KeyvalText
          size={20}
          weight={600}
        >{`Add ${name} Destination`}</KeyvalText>
      </HeaderTitleWrapper>
    </SetupHeaderWrapper>
  );
}
