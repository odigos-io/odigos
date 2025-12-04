import React, { type FC } from 'react';
import { useStatusStore } from '@/store';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { useConfig, useDescribe, useTokenCRUD } from '@/hooks';
import { OtherStatusType, StatusType } from '@odigos/ui-kit/types';
import { Header, Status, Tooltip } from '@odigos/ui-kit/components';
import { Badge, Header as V2Header } from '@odigos/ui-kit/components/v2';
import { NotificationManager, SlackInvite, SystemOverview, ToggleDarkMode } from '@odigos/ui-kit/containers';

interface OverviewHeaderProps {
  v2?: boolean;
}

const OverviewHeader: FC<OverviewHeaderProps> = ({ v2 = false }) => {
  const { status, title, message } = useStatusStore();

  const { isReadonly } = useConfig();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();

  if (v2) {
    return (
      <V2Header
        left={[
          <OdigosLogoText key='logo' size={150} />,
          <Badge key='status' status={status} label={message} />,
          isReadonly && (
            <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
              <Badge status={OtherStatusType.Disabled} label='Read Only' />
            </Tooltip>
          ),
        ]}
        right={[
          <NotificationManager key='notifs' />,
          <SystemOverview key='system-overview' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />,
          <SlackInvite key='slack' />,
        ]}
      />
    );
  }

  return (
    <Header
      left={[
        <OdigosLogoText key='logo' size={150} />,
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
        <SlackInvite key='slack' />,
      ]}
    />
  );
};

export { OverviewHeader };
