'use client';

import React, { type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import styled from 'styled-components';
import { OverviewHeader } from '@/components';
import { Navbar } from '@odigos/ui-kit/components';
import { ToastList } from '@odigos/ui-kit/containers';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { getNavbarIcons, INITIAL_CONTEXT } from '@/utils';
import { useConfig, useSSE, useTokenTracker } from '@/hooks';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';

const ViewportColumn = styled(FlexColumn)`
  height: 100vh;
  overflow: hidden;
`;

const ContentRow = styled(FlexRow)`
  flex: 1;
  min-height: 0;
`;

function InnerLayout({ children }: PropsWithChildren) {
  useSSE();
  useTokenTracker();

  const router = useRouter();
  const pathname = usePathname();
  const { config } = useConfig();

  return (
    <OdigosProvider platformType={config?.platformType ?? INITIAL_CONTEXT.platformType} tier={config?.tier ?? INITIAL_CONTEXT.tier} version={config?.odigosVersion || INITIAL_CONTEXT.version}>
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
  return (
    <ErrorBoundary>
      <OdigosApiAdapter>
        <InnerLayout>{children}</InnerLayout>
      </OdigosApiAdapter>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
