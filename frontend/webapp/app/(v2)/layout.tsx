'use client';

import React, { type PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import styled from 'styled-components';
import { OverviewHeader } from '@/components';
import { Navbar } from '@odigos/ui-kit/components/v2';
import { ToastList } from '@odigos/ui-kit/containers';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { PlatformType, Tier } from '@odigos/ui-kit/types';
import { getNavbarIcons, INITIAL_CONTEXT } from '@/utils';
import { useConfig, useSSE, useTokenTracker } from '@/hooks';
import type { OperationContext } from '@odigos/ui-kit/contexts/odigos-api';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';

const ViewportColumn = styled(FlexColumn)`
  height: 100vh;
  overflow: hidden;
`;

const ContentRow = styled(FlexRow)`
  flex: 1;
  min-height: 0;
`;

/**
 * Reads `GET_CONFIG` (via Apollo, which is mounted by the parent
 * `<OdigosApiAdapter>`) and reports the derived operation context up to
 * the layout. The layout re-renders the adapter with the resolved
 * context — the `OdigosApiProvider` is memoized on `httpUrl` so the
 * Apollo client is preserved across context changes (no remount).
 */
function ConfigSync({ onContext }: { onContext: (ctx: OperationContext) => void }) {
  const { config, isReadonly } = useConfig();

  useEffect(() => {
    onContext({
      platformType: (config?.platformType as PlatformType) ?? PlatformType.K8s,
      tier: (config?.tier as Tier) ?? Tier.Community,
      version: config?.odigosVersion || 'v0.0.0',
      isReadonly,
    });
  }, [config?.platformType, config?.tier, config?.odigosVersion, isReadonly, onContext]);

  return null;
}

function InnerLayout({ children, context }: PropsWithChildren<{ context: OperationContext }>) {
  useSSE();
  useTokenTracker();

  const router = useRouter();
  const pathname = usePathname();

  return (
    <OdigosProvider platformType={context.platformType} tier={context.tier} version={context.version}>
      <ViewportColumn $gap={0}>
        <OverviewHeader />
        <ContentRow $gap={0}>
          <Navbar height='calc(100vh - 60px)' icons={getNavbarIcons(router, pathname)} />
          {children}
        </ContentRow>
      </ViewportColumn>

      <ToastList />
    </OdigosProvider>
  );
}

function OverviewLayout({ children }: PropsWithChildren) {
  const [context, setContext] = useState<OperationContext>(INITIAL_CONTEXT);
  // Stable reference for the ConfigSync callback so its `useEffect`
  // doesn't re-fire on every parent render.
  const onContext = useMemo(() => (ctx: OperationContext) => setContext(ctx), []);

  return (
    <ErrorBoundary>
      <OdigosApiAdapter context={context}>
        <ConfigSync onContext={onContext} />
        <InnerLayout context={context}>{children}</InnerLayout>
      </OdigosApiAdapter>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
