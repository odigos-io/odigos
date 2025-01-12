import React from 'react';
import theme from '@/styles/theme';
import { FlexRow } from '@/styles';
import { SLACK_LINK } from '@/utils';
import styled from 'styled-components';
import { PlatformTypes } from '@/types';
import { PlatformTitle } from './cp-title';
import { NotificationManager } from '@/components';
import { OdigosLogoText, SlackLogo, TerminalIcon } from '@/assets';
import { ConnectionStatus, IconButton } from '@/reuseable-components';
import { DRAWER_OTHER_TYPES, useConnectionStore, useDrawerStore } from '@/store';

interface MainHeaderProps {}

const HeaderContainer = styled(FlexRow)`
  width: 100%;
  padding: 12px 0;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid rgba(249, 249, 249, 0.16);
`;

const AlignLeft = styled(FlexRow)`
  margin-right: auto;
  margin-left: 32px;
  gap: 16px;
`;

const AlignRight = styled(FlexRow)`
  margin-left: auto;
  margin-right: 32px;
  gap: 12px;
`;

export const MainHeader: React.FC<MainHeaderProps> = () => {
  const { setSelectedItem } = useDrawerStore();
  const { connecting, active, title, message } = useConnectionStore();

  const handleClickCli = () => setSelectedItem({ type: DRAWER_OTHER_TYPES.ODIGOS_CLI, id: DRAWER_OTHER_TYPES.ODIGOS_CLI });
  const handleClickSlack = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <HeaderContainer>
      <AlignLeft>
        <OdigosLogoText size={80} />
        <PlatformTitle type={PlatformTypes.K8S} />
        {!connecting && <ConnectionStatus title={title} subtitle={message} isActive={active} />}
      </AlignLeft>

      <AlignRight>
        <IconButton onClick={handleClickCli} tooltip='Odigos CLI' withPing pingColor={theme.colors.majestic_blue}>
          <TerminalIcon size={18} />
        </IconButton>

        <NotificationManager />

        <IconButton onClick={handleClickSlack} tooltip='Join our Slack community'>
          <SlackLogo />
        </IconButton>
      </AlignRight>
    </HeaderContainer>
  );
};
