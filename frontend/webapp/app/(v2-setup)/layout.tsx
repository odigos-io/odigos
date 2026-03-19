'use client';

import React, { type PropsWithChildren, useEffect } from 'react';
import { useConfig } from '@/hooks';
import { useDarkMode } from '@odigos/ui-kit/store';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { ErrorBoundary } from '@odigos/ui-kit/components';

function Layout({ children }: PropsWithChildren) {
  // TODO: remove this after migration to v2
  const { darkMode, setDarkMode } = useDarkMode();
  useEffect(() => {
    if (!darkMode) setDarkMode(true);
    document.body.style.backgroundColor = '#151618';
  }, []);

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
