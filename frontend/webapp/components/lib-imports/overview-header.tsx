import React, { useMemo, useState } from 'react';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { OtherStatusType } from '@odigos/ui-kit/types';
import { StatusKeys, useStatusStore } from '../../store';
import { Header, Tooltip } from '@odigos/ui-kit/components';
import { SystemDrawer } from '@odigos/ui-kit/containers/v2';
import { useConfig, useDescribe, useDiagnose } from '@/hooks';
import { OdigosLogoText, TerminalIcon } from '@odigos/ui-kit/icons';
import { NotificationManager, SlackInvite, ToggleDarkMode } from '@odigos/ui-kit/containers';
import { IconButton, Badge as V2Badge, Header as V2Header } from '@odigos/ui-kit/components/v2';

const OverviewHeader = ({ v2 }: { v2?: boolean }) => {
  const { isReadonly } = useConfig();
  const { downloadDiagnose } = useDiagnose();
  const { fetchDescribeOdigos } = useDescribe();

  const [isSystemDrawerOpen, setIsSystemDrawerOpen] = useState(false);
  const toggleSystemDrawer = () => setIsSystemDrawerOpen((prev) => !prev);

  const tokenStatus = useStatusStore((state) => state[StatusKeys.Token]);
  const backendStatus = useStatusStore((state) => state[StatusKeys.Backend]);
  const instrumentationStatus = useStatusStore((state) => state[StatusKeys.Instrumentation]);

  const left = useMemo(() => {
    const arr = [<OdigosLogoText key='logo' size={150} />];

    if (tokenStatus) {
      arr.push(
        <Tooltip key='token-status' text={tokenStatus.tooltip}>
          <V2Badge {...tokenStatus} />
        </Tooltip>,
      );
    }
    if (backendStatus) {
      arr.push(
        <Tooltip key='backend-status' text={backendStatus.tooltip}>
          <V2Badge {...backendStatus} />
        </Tooltip>,
      );
    }
    if (instrumentationStatus) {
      arr.push(
        <Tooltip key='instrumentation-status' text={instrumentationStatus.tooltip}>
          <V2Badge {...instrumentationStatus} />
        </Tooltip>,
      );
    }
    if (isReadonly) {
      arr.push(
        <Tooltip key='readonly' text={FORM_ALERTS.READONLY_WARNING}>
          <V2Badge status={OtherStatusType.Disabled} label='Read Only' />
        </Tooltip>,
      );
    }

    return arr;
  }, [v2, tokenStatus?.label, backendStatus?.label, instrumentationStatus?.label, isReadonly]);

  const right = useMemo(() => {
    const arr = [<NotificationManager key='notification-manager' />];
    if (!v2) arr.unshift(<ToggleDarkMode key='toggle-theme' />);
    arr.push(<IconButton key='system-drawer' icon={TerminalIcon} onClick={toggleSystemDrawer} />);
    arr.push(...[<SlackInvite key='slack-invite' />]);

    return arr;
  }, [v2]);

  if (v2) {
    return (
      <>
        <V2Header left={left} right={right} />
        <SystemDrawer isOpen={isSystemDrawerOpen} onClose={toggleSystemDrawer} fetchDescribeOdigos={fetchDescribeOdigos} downloadDiagnose={downloadDiagnose} />
      </>
    );
  }

  return (
    <>
      <Header left={left} right={right} />
      <SystemDrawer isOpen={isSystemDrawerOpen} onClose={toggleSystemDrawer} fetchDescribeOdigos={fetchDescribeOdigos} downloadDiagnose={downloadDiagnose} />
    </>
  );
};

export { OverviewHeader };
