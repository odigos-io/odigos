import React from 'react';
import { useConfig } from '@/hooks';
import { useStatusStore } from '@/store';
import { Theme } from '@odigos/ui-theme';
import { PlatformTitle } from './cp-title';
import { FORM_ALERTS, SLACK_LINK } from '@/utils';
import styled, { useTheme } from 'styled-components';
import { NOTIFICATION_TYPE, PLATFORM_TYPE } from '@odigos/ui-utils';
import { OdigosLogoText, SlackLogo, TerminalIcon } from '@odigos/ui-icons';
import { FlexRow, IconButton, Status, ToggleDarkMode, Tooltip } from '@odigos/ui-components';
import { DRAWER_OTHER_TYPES, NotificationManager, useDarkModeStore, useDrawerStore } from '@odigos/ui-containers';

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
  const { setDrawerType } = useDrawerStore();
  const { status, title, message } = useStatusStore();
  const { darkMode, setDarkMode } = useDarkModeStore();

  const handleClickCli = () => setDrawerType(DRAWER_OTHER_TYPES.ODIGOS_CLI);
  const handleClickSlack = () => window.open(SLACK_LINK, '_blank', 'noopener noreferrer');

  return (
    <HeaderContainer>
      <AlignLeft>
        <OdigosLogoText size={80} />
        <PlatformTitle type={PLATFORM_TYPE.K8S} />
        <Status status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />
        {config?.readonly && (
          <Tooltip text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
          </Tooltip>
        )}
      </AlignLeft>

      <AlignRight>
        <ToggleDarkMode darkMode={darkMode} setDarkMode={setDarkMode} />
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
