import React from 'react';
import { useStatusStore } from '@/store';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { getPlatformLabel } from '@odigos/ui-kit/functions';
import { useConfig, useDescribe, useTokenCRUD } from '@/hooks';
import { PlatformType, StatusType } from '@odigos/ui-kit/types';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { ComputePlatformSelect, NotificationManager, SlackInvite, SystemOverview, ToggleDarkMode } from '@odigos/ui-kit/containers';

const OverviewHeader = () => {
  const { status, title, message } = useStatusStore();

  const { isReadonly } = useConfig();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();

  return (
    <Header
      left={[
        <OdigosLogoText key='logo' size={100} />,
        <ComputePlatformSelect
          key='cp-select'
          selected={{
            id: 'default',
            name: getPlatformLabel(PlatformType.K8s),
            type: PlatformType.K8s,
            connectionStatus: StatusType.Success,
          }}
          connections={[]}
          onSelect={() => {}}
          onViewAll={() => {}}
        />,
        <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
        isReadonly && (
          <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={StatusType.Info} title='Read Only' size={14} family='primary' withIcon withBackground />
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
