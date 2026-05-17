'use client';

import React, { type PropsWithChildren } from 'react';
import { useConfig } from '@/hooks';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { ErrorBoundary } from '@odigos/ui-kit/components';

function Layout({ children }: PropsWithChildren) {
  const { config } = useConfig();

  return (
    <ErrorBoundary>
      <OdigosProvider platformType={config?.platformType} tier={config?.tier} version={config?.odigosVersion || ''}>
        {children}
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default Layout;
