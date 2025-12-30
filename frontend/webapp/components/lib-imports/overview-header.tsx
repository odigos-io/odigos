import React, { useMemo } from 'react';
import { useStatusStore } from '../../store';
import { OdigosLogoText } from '@odigos/ui-kit/icons';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { OtherStatusType } from '@odigos/ui-kit/types';
import { Header, Tooltip } from '@odigos/ui-kit/components';
import { useConfig, useDescribe, useTokenCRUD } from '@/hooks';
import { Badge as V2Badge, Header as V2Header } from '@odigos/ui-kit/components/v2';
import { NotificationManager, SlackInvite, SystemOverview, ToggleDarkMode } from '@odigos/ui-kit/containers';

const OverviewHeader = ({ v2 }: { v2?: boolean }) => {
  const { isReadonly } = useConfig();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();
  const { status, message, leftIcon } = useStatusStore();

  const left = useMemo(() => {
    const arr = [<OdigosLogoText key='logo' size={150} />];

    if (message) {
      arr.push(<V2Badge key='status' status={status} label={message} leftIcon={leftIcon} />);
    }
    if (isReadonly) {
      arr.push(
        <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
          <V2Badge status={OtherStatusType.Disabled} label='Read Only' />
        </Tooltip>,
      );
    }

    return arr;
  }, [v2, message, leftIcon, status, isReadonly]);

  const right = useMemo(() => {
    const arr = [<NotificationManager key='notification-manager' />];
    if (!v2) arr.unshift(<ToggleDarkMode key='toggle-theme' />);
    arr.push(<SystemOverview key='system-overview' tokens={tokens} saveToken={updateToken} fetchDescribeOdigos={fetchDescribeOdigos} />);
    arr.push(...[<SlackInvite key='slack-invite' />]);

    return arr;
  }, [v2, tokens]);

  if (v2) {
    return <V2Header left={left} right={right} />;
  }

  return <Header left={left} right={right} />;
};

export { OverviewHeader };
