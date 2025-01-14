import React from 'react';
import { FlexRow } from '@/styles';
import { SLACK_LINK } from '@/utils';
import { PlatformTitle } from './cp-title';
import { NotificationManager } from '@/components';
import styled, { useTheme } from 'styled-components';
import { NOTIFICATION_TYPE, PlatformTypes } from '@/types';
import { ConnectionStatus, IconButton } from '@/reuseable-components';
import { LightOffIcon, LightOnIcon, OdigosLogoText, SlackLogo, TerminalIcon } from '@/assets';
import { DRAWER_OTHER_TYPES, useConnectionStore, useDarkModeStore, useDrawerStore } from '@/store';

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
  const theme = useTheme();
  const { setSelectedItem } = useDrawerStore();
  const { darkMode, setDarkMode } = useDarkModeStore();
  const { title, message, sseConnecting, sseStatus, tokenExpired, tokenExpiring } = useConnectionStore();

  const handleClickCli = () => setSelectedItem({ type: DRAWER_OTHER_TYPES.ODIGOS_CLI, id: DRAWER_OTHER_TYPES.ODIGOS_CLI });
  const handleClickSlack = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <HeaderContainer>
      <AlignLeft>
        <OdigosLogoText size={80} />
        <PlatformTitle type={PlatformTypes.K8S} />
        {!sseConnecting && <ConnectionStatus title={title} subtitle={message} status={tokenExpired ? NOTIFICATION_TYPE.ERROR : tokenExpiring ? NOTIFICATION_TYPE.WARNING : sseStatus} />}
      </AlignLeft>

      <AlignRight>
        <IconButton onClick={handleClickCli} tooltip='Odigos CLI' withPing pingColor={theme.colors.majestic_blue}>
          <TerminalIcon size={18} />
        </IconButton>

        <NotificationManager />

        <IconButton onClick={() => setDarkMode(!darkMode)} tooltip={darkMode ? 'Light Mode' : 'Dark Mode'}>
          {darkMode ? <LightOffIcon size={18} /> : <LightOnIcon size={18} />}
        </IconButton>

        <IconButton onClick={handleClickSlack} tooltip='Join our Slack community'>
          <SlackLogo />
        </IconButton>
      </AlignRight>
    </HeaderContainer>
  );
};
