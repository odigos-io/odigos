import React, { useMemo, useState } from 'react';
import { TerminalIcon } from '@odigos/ui-kit/icons';
import { Tooltip } from '@odigos/ui-kit/components';
import { StatusKeys, useStatusStore } from '../store';
import { FORM_ALERTS } from '@odigos/ui-kit/constants';
import { OtherStatusType } from '@odigos/ui-kit/types';
import { SystemDrawer } from '@odigos/ui-kit/containers/v2';
import { OdigosLogoTextByTier } from '@odigos/ui-kit/snippets/v2';
import { useConfig, useDescribe, useDiagnose, useTokenCRUD } from '@/hooks';
import { IconButton, Badge as V2Badge, Header as V2Header } from '@odigos/ui-kit/components/v2';

export const OverviewHeader = () => {
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
        <OdigosLogoTextByTier size={200} />
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
  }, [isReadonly, backendStatus, instrumentationStatus]);

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
    arr.push(
      <div key='system-drawer' data-id='system-drawer'>
        <IconButton icon={TerminalIcon} onClick={toggleSystemDrawer} />
      </div>,
    );

    return arr;
  }, [tokenStatus]);

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
};
