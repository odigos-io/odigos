'use client';

import React, { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { useRouter } from 'next/navigation';
import OdigosApiAdapter from '@/lib/odigos-api-adapter';
import { InstallationStatus } from '@odigos/ui-kit/types';
import { CenterThis, Loader } from '@odigos/ui-kit/components';

/**
 * The root `/` page reads `config.installationStatus` and redirects
 * to either `/onboarding` (fresh install) or `/overview` (already
 * installed). `useConfig` goes through the kit's `useApiQuery`, which
 * requires `<OdigosApiProvider>` mounted above it — `(v2)` and
 * `(setup)` route groups each have their own adapter, but the bare
 * root page does not. We wrap inline so this page can also fetch
 * config without crashing the kit's "called outside of provider"
 * guard.
 */
function Redirect() {
  const router = useRouter();
  const { config } = useConfig();

  useEffect(() => {
    if (config) {
      const { installationStatus, readonly } = config;
      const redirectTo = !readonly && installationStatus === InstallationStatus.New ? ROUTES.ONBOARDING : ROUTES.OVERVIEW;

      router.push(redirectTo);
    }
  }, [config]);

  return (
    <CenterThis style={{ height: '100%' }}>
      <Loader withSpinnerOld scaleSpinnerOld={2} />
    </CenterThis>
  );
}

export default function App() {
  return (
    <OdigosApiAdapter>
      <Redirect />
    </OdigosApiAdapter>
  );
}
