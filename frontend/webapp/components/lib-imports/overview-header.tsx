import React from 'react';
import { useStatusStore } from '@/store';
import { StatusType } from '@odigos/ui-kit/types';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { useConfig, useDescribe, useOdigosConfigCRUD, useTokenCRUD } from '@/hooks';
import { NotificationManager, SlackInvite, SystemOverview, SystemSettings, ToggleDarkMode } from '@odigos/ui-kit/containers';

const OverviewHeader = () => {
  const { status, title, message } = useStatusStore();

  const { isReadonly, installationMethod } = useConfig();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();
  const { fetchOdigosConfig, updateOdigosConfig } = useOdigosConfigCRUD();

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
        <NotificationManager key='notifs' />,
        <SystemOverview key='system-overview' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />,
        <SystemSettings key='system-settings' installationMethod={installationMethod} fetchSettings={fetchOdigosConfig} onSave={updateOdigosConfig} />,
        <SlackInvite key='slack' />,
      ]}
    />
  );
};

export { OverviewHeader };
