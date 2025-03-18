import React from 'react';
import { useStatusStore } from '@/store';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { getPlatformLabel } from '@odigos/ui-kit/functions';
import { useConfig, useDescribe, useTokenCRUD } from '@/hooks';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { STATUS_TYPE, PLATFORM_TYPE } from '@odigos/ui-kit/types';
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
            name: getPlatformLabel(PLATFORM_TYPE.K8S),
            type: PLATFORM_TYPE.K8S,
            connectionStatus: STATUS_TYPE.SUCCESS,
          }}
          connections={[]}
          onSelect={() => {}}
          onViewAll={() => {}}
        />,
        <Status key='status' status={status} title={title} subtitle={message} size={14} family='primary' withIcon withBackground />,
        isReadonly && (
          <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
            <Status status={STATUS_TYPE.INFO} title='Read Only' size={14} family='primary' withIcon withBackground />
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
