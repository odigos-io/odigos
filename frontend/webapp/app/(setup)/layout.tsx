'use client';

import React, { type PropsWithChildren } from 'react';
import { useConfig } from '@/hooks';
import { INITIAL_CONTEXT } from '@/utils';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { ErrorBoundary } from '@odigos/ui-kit/components';

function InnerLayout({ children }: PropsWithChildren) {
  const { config } = useConfig();

  return (
    <OdigosProvider platformType={config?.platformType ?? INITIAL_CONTEXT.platformType} tier={config?.tier ?? INITIAL_CONTEXT.tier} version={config?.odigosVersion || INITIAL_CONTEXT.version}>
      {children}
    </OdigosProvider>
  );
}

function Layout({ children }: PropsWithChildren) {
  return (
    <ErrorBoundary>
      <OdigosApiAdapter>
        <InnerLayout>{children}</InnerLayout>
      </OdigosApiAdapter>
    </ErrorBoundary>
  );
}

export default Layout;
