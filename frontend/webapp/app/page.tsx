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
      const redirectTo = !readonly && installationStatus === InstallationStatus.New ? ROUTES.CHOOSE_SOURCES : ROUTES.OVERVIEW;

      router.push(redirectTo);
    }
  }, [config]);

  return (
    <CenterThis style={{ height: '100%' }}>
      <FadeLoader scale={2} />
    </CenterThis>
  );
}
