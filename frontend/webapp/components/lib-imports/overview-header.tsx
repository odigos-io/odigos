import React from 'react';
import { useStatusStore } from '@/store';
import { StatusType } from '@odigos/ui-kit/types';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useConfig, useDescribe, useTokenCRUD } from '@/hooks';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { NotificationManager, SlackInvite, SystemOverview, ToggleDarkMode } from '@odigos/ui-kit/containers';

const OverviewHeader = () => {
  const { status, title, message } = useStatusStore();

  const { isReadonly } = useConfig();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();

  return (
    <Header
      left={[
        <OdigosLogoText key='logo' size={100} />,
        <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
        isReadonly && (
          <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={StatusType.Info} title='Read Only' size={14} family='primary' withIcon withBackground />
          </Tooltip>
        ),
      ]}
      right={[
        <ToggleDarkMode key='toggle-theme' />,
        <SystemOverview key='system-overview' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />,
        <NotificationManager key='notifs' />,
        <SlackInvite key='slack' />,
      ]}
    />
  );
};

export { OverviewHeader };
