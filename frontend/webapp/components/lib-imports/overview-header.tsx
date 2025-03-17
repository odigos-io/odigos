import React from 'react';
import { useStatusStore } from '@/store';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { getPlatformLabel } from '@odigos/ui-kit/functions';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { useConfig, useDescribeOdigos, useTokenCRUD } from '@/hooks';
import { NOTIFICATION_TYPE, PLATFORM_TYPE } from '@odigos/ui-kit/types';
import { ComputePlatformSelect, NotificationManager, SlackInvite, SystemOverview, ToggleDarkMode } from '@odigos/ui-kit/containers';

const OverviewHeader = () => {
  const { status, title, message } = useStatusStore();

  const { data: config } = useConfig();
  const { tokens, updateToken } = useTokenCRUD();
  const { fetchDescribeOdigos } = useDescribeOdigos();

  return (
    <Header
      left={[
        <OdigosLogoText key='logo' size={100} />,
        <ComputePlatformSelect
          key='cp-select'
          selected={{
            id: 'default',
            name: getPlatformLabel(PLATFORM_TYPE.K8S),
            type: PLATFORM_TYPE.K8S,
            connectionStatus: NOTIFICATION_TYPE.SUCCESS,
          }}
          connections={[]}
          onSelect={() => {}}
          onViewAll={() => {}}
        />,
        <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
        config?.readonly && (
          <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={NOTIFICATION_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
          </Tooltip>
        ),
      ]}
      right={[
        <ToggleDarkMode key='toggle-theme' />,
        <NotificationManager key='notifs' />,
        <SystemOverview key='cli' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />,
        <SlackInvite key='slack' />,
      ]}
    />
  );
};

export { OverviewHeader };
