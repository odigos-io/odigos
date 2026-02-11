'use client';

import React, { type PropsWithChildren, useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { getNavbarIcons } from '@/utils';
import { OverviewHeader } from '@/components';
import { useDarkMode } from '@odigos/ui-kit/store';
import { Navbar } from '@odigos/ui-kit/components/v2';
import { ToastList } from '@odigos/ui-kit/containers';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { useConfig, useSSE, useTokenTracker } from '@/hooks';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  // TODO: remove this after migration to v2
  const { darkMode, setDarkMode } = useDarkMode();
  useEffect(() => {
    if (!darkMode) setDarkMode(true);
    document.body.style.backgroundColor = '#151618';
  }, []);

  const router = useRouter();
  const pathname = usePathname();
  const { config } = useConfig();

  return (
    <ErrorBoundary>
      <OdigosProvider platformType={config?.platformType} tier={config?.tier} version={config?.odigosVersion || ''}>
        <FlexColumn $gap={0}>
          <OverviewHeader v2 />
          <FlexRow $gap={0}>
            <Navbar height='calc(100vh - 60px)' icons={getNavbarIcons(router, pathname)} />
            {children}
          </FlexRow>
        </FlexColumn>

        <ToastList />
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
