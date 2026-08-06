'use client';

import React, { type PropsWithChildren } from 'react';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { ToastList } from '@odigos/ui-kit/containers';
import { ErrorBoundary } from '@odigos/ui-kit/components';

/**
 * Trace correlations lives outside the `(v2)` route group so it keeps its
 * standalone layout, but it still needs Apollo for the temp hooks that call
 * `useQuery` / `useMutation` directly.
 */
function TraceCorrelationsLayout({ children }: PropsWithChildren) {
  return (
    <ErrorBoundary>
      <OdigosApiAdapter>
        {children}
        <ToastList />
      </OdigosApiAdapter>
    </ErrorBoundary>
  );
}

export default TraceCorrelationsLayout;
