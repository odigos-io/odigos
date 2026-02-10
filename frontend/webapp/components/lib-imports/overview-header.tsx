import React, { useMemo, useState } from 'react';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { OtherStatusType } from '@odigos/ui-kit/types';
import { StatusKeys, useStatusStore } from '../../store';
import { Header, Tooltip } from '@odigos/ui-kit/components';
import { SystemDrawer } from '@odigos/ui-kit/containers/v2';
import { OdigosLogoText, TerminalIcon } from '@odigos/ui-kit/icons';
import { useConfig, useDescribe, useDiagnose, useTokenCRUD } from '@/hooks';
import { NotificationManager, SlackInvite, ToggleDarkMode } from '@odigos/ui-kit/containers';
import { IconButton, Badge as V2Badge, Header as V2Header } from '@odigos/ui-kit/components/v2';

const OverviewHeader = ({ v2 }: { v2?: boolean }) => {
  const { isReadonly } = useConfig();
  const { downloadDiagnose } = useDiagnose();
  const { fetchDescribeOdigos } = useDescribe();
  const { tokens, updateToken } = useTokenCRUD();

  const [isSystemDrawerOpen, setIsSystemDrawerOpen] = useState(false);
  const toggleSystemDrawer = () => setIsSystemDrawerOpen((prev) => !prev);

  const tokenStatus = useStatusStore((state) => state[StatusKeys.Token]);
  const backendStatus = useStatusStore((state) => state[StatusKeys.Backend]);
  const instrumentationStatus = useStatusStore((state) => state[StatusKeys.Instrumentation]);

  const left = useMemo(() => {
    const arr = [
      <div key='logo' data-id='logo'>
        <OdigosLogoText size={150} />
      </div>,
    ];

    if (isReadonly) {
      arr.push(
        <div key='readonly' data-id='readonly'>
          <Tooltip text={FORM_ALERTS.READONLY_WARNING}>
            <V2Badge status={OtherStatusType.Disabled} label='Read Only' />
          </Tooltip>
        </div>,
      );
    }
    if (backendStatus) {
      arr.push(
        <div key='backend-status' data-id='backend-status'>
          <Tooltip text={backendStatus.tooltip}>
            <V2Badge {...backendStatus} />
          </Tooltip>
        </div>,
      );
    }
    if (instrumentationStatus) {
      arr.push(
        <div key='instrumentation-status' data-id='instrumentation-status'>
          <Tooltip text={instrumentationStatus.tooltip}>
            <V2Badge {...instrumentationStatus} />
          </Tooltip>
        </div>,
      );
    }

    return arr;
  }, [v2, isReadonly, backendStatus?.label, instrumentationStatus?.label]);

  const right = useMemo(() => {
    const arr = [];

    if (tokenStatus) {
      arr.push(
        <div key='token-status' data-id='token-status'>
          <Tooltip text={tokenStatus.tooltip}>
            <V2Badge {...tokenStatus} />
          </Tooltip>
        </div>,
      );
    }

    if (!v2)
      arr.push(
        <div key='toggle-theme' data-id='toggle-theme'>
          <ToggleDarkMode />
        </div>,
      );
    arr.push(
      <div key='notifications' data-id='notifications'>
        <NotificationManager />
      </div>,
    );
    arr.push(
      <div key='system-drawer' data-id='system-drawer'>
        <IconButton icon={TerminalIcon} onClick={toggleSystemDrawer} />
      </div>,
    );
    arr.push(
      ...[
        <div key='slack-invite' data-id='slack-invite'>
          <SlackInvite />
        </div>,
      ],
    );

    return arr;
  }, [v2, tokenStatus?.label]);

  if (v2) {
    return (
      <>
        <V2Header left={left} right={right} />
        <SystemDrawer
          isOpen={isSystemDrawerOpen}
          onClose={toggleSystemDrawer}
          fetchDescribeOdigos={fetchDescribeOdigos}
          downloadDiagnose={downloadDiagnose}
          token={tokens[0]}
          updateToken={updateToken}
        />
      </>
    );
  }

  return (
    <>
      <Header left={left} right={right} />
      <SystemDrawer
        isOpen={isSystemDrawerOpen}
        onClose={toggleSystemDrawer}
        fetchDescribeOdigos={fetchDescribeOdigos}
        downloadDiagnose={downloadDiagnose}
        token={tokens[0]}
        updateToken={updateToken}
      />
    </>
  );
};

export { OverviewHeader };
