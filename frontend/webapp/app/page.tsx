'use client';

import { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { usePathname, useRouter } from 'next/navigation';
import { InstallationStatus } from '@/types';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';

export default function App() {
  const router = useRouter();
  const pathname = usePathname();
  const { config } = useConfig();

  useEffect(() => {
    if (pathname === ROUTES.ROOT && config) {
      const { installationStatus, readonly } = config;

      if (installationStatus === InstallationStatus.New && !readonly) {
        router.push(ROUTES.CHOOSE_SOURCES);
      } else {
        router.push(ROUTES.OVERVIEW);
      }
    }
  }, [pathname, config]);

  return (
    <CenterThis style={{ height: '100%' }}>
      <FadeLoader scale={2} />
    </CenterThis>
  );
}
