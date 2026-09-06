'use client';

import React, { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { useRouter } from 'next/navigation';
import { Insights } from '@odigos/ui-kit/containers';
import { CenterThis, Loader } from '@odigos/ui-kit/components';

// The nav entry is already hidden when insights are disabled, so this guard
// only catches direct navigation to `/insights`.
export default function Page() {
  const router = useRouter();
  const { config } = useConfig();

  useEffect(() => {
    if (config && !config.insightsEnabled) router.replace(ROUTES.OVERVIEW);
  }, [config, router]);

  if (!config?.insightsEnabled) {
    return (
      <CenterThis style={{ height: '100%' }}>
        <Loader withSpinnerOld scaleSpinnerOld={2} />
      </CenterThis>
    );
  }

  return <Insights />;
}
