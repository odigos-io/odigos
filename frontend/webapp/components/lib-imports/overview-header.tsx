import React from 'react';
import Theme from '@odigos/ui-theme';
import { SLACK_LINK } from '@/utils';
import { useStatusStore } from '@/store';
import { OdigosLogoText, SlackLogo } from '@odigos/ui-icons';
import { useConfig, useDescribeOdigos, useTokenCRUD } from '@/hooks';
import { CliDrawer, NotificationManager } from '@odigos/ui-containers';
import { FORM_ALERTS, NOTIFICATION_TYPE, PLATFORM_TYPE } from '@odigos/ui-utils';
import { Header, IconButton, PlatformSelect, Status, Tooltip } from '@odigos/ui-components';

const OverviewHeader = () => {
  const { status, title, message } = useStatusStore();

  const { data: config } = useConfig();
  const { tokens, updateToken } = useTokenCRUD();
  const { fetchDescribeOdigos } = useDescribeOdigos();

  return (
    <Header
      left={[
        <OdigosLogoText key='logo' size={100} />,
        <PlatformSelect key='platform' type={PLATFORM_TYPE.K8S} />,
        <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
        config?.readonly && (
          <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
          </Tooltip>
        ),
      ]}
      right={[
        <Theme.ToggleDarkMode key='toggle-theme' />,
        <NotificationManager key='notifs' />,
        <CliDrawer key='cli' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />,
        <IconButton key='slack' onClick={() => window.open(SLACK_LINK, '_blank', 'noopener noreferrer')} tooltip='Join our Slack community'>
          <SlackLogo />
        </IconButton>,
      ]}
    />
  );
};

export default OverviewHeader;
