'use client';

import React, { type PropsWithChildren, useEffect, useMemo, useState } from 'react';
import { useConfig } from '@/hooks';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { ErrorBoundary } from '@odigos/ui-kit/components';
import { PlatformType, Tier } from '@odigos/ui-kit/types';
import type { OperationContext } from '@odigos/ui-kit/contexts/odigos-api';

const INITIAL_CONTEXT: OperationContext = {
  platformType: PlatformType.K8s,
  tier: Tier.Community,
  version: 'v0.0.0',
};

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

function Layout({ children }: PropsWithChildren) {
  const [context, setContext] = useState<OperationContext>(INITIAL_CONTEXT);
  const onContext = useMemo(() => (ctx: OperationContext) => setContext(ctx), []);

  return (
    <ErrorBoundary>
      <OdigosApiAdapter context={context}>
        <ConfigSync onContext={onContext} />
        <OdigosProvider platformType={context.platformType} tier={context.tier} version={context.version}>
          {children}
        </OdigosProvider>
      </OdigosApiAdapter>
    </ErrorBoundary>
  );
}

export default Layout;
