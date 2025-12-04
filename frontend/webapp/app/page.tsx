'use client';

import { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { useRouter } from 'next/navigation';
import { InstallationStatus } from '@/types';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';

export default function App() {
  const router = useRouter();
  const { config } = useConfig();

  useEffect(() => {
    if (config) {
      const { installationStatus, readonly } = config;

      if (installationStatus === InstallationStatus.New && !readonly) {
        // TODO: fix this (always redirecting to the choose sources page in tests...)
        // router.push(ROUTES.CHOOSE_SOURCES);
      } else {
        router.push(ROUTES.OVERVIEW);
      }
    }
  }, [config]);

  return (
    <CenterThis style={{ height: '100%' }}>
      <FadeLoader scale={2} />
    </CenterThis>
  );
}
