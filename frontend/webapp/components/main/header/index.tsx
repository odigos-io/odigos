import React from 'react';
import { useConfig } from '@/hooks';
import { PlatformTitle } from './cp-title';
import { FORM_ALERTS, SLACK_LINK } from '@/utils';
import styled, { useTheme } from 'styled-components';
import { NOTIFICATION_TYPE, PlatformTypes } from '@/types';
import { NotificationManager, ToggleDarkMode } from '@/components';
import { DRAWER_OTHER_TYPES, useDrawerStore, useStatusStore } from '@/store';
import { FlexRow, IconButton, OdigosLogoText, SlackLogo, Status, TerminalIcon, Theme, Tooltip } from '@odigos/ui-components';

interface MainHeaderProps {}

const HeaderContainer = styled(FlexRow)`
  width: 100%;
  padding: 12px 0;
  background-color: ${({ theme }) => theme.colors.dark_grey};
  border-bottom: 1px solid ${({ theme }) => theme.colors.border + Theme.hexPercent['050']};
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
  const { data: config } = useConfig();
  const { setSelectedItem } = useDrawerStore();
  const { status, title, message } = useStatusStore();

  const handleClickCli = () => setSelectedItem({ type: DRAWER_OTHER_TYPES.ODIGOS_CLI, id: DRAWER_OTHER_TYPES.ODIGOS_CLI });
  const handleClickSlack = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <HeaderContainer>
      <AlignLeft>
        <OdigosLogoText size={80} />
        <PlatformTitle type={PlatformTypes.K8S} />
        <Status status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />
        {config?.readonly && (
          <Tooltip text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
          </Tooltip>
        )}
      </AlignLeft>

      <AlignRight>
        <ToggleDarkMode />
        <NotificationManager />

        <IconButton onClick={handleClickCli} tooltip='Odigos CLI' withPing pingColor={theme.colors.majestic_blue}>
          <TerminalIcon size={18} />
        </IconButton>
        <IconButton onClick={handleClickSlack} tooltip='Join our Slack community'>
          <SlackLogo />
        </IconButton>
      </AlignRight>
    </HeaderContainer>
  );
};
