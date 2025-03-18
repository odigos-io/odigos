'use client';

import { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { useRouter } from 'next/navigation';
import { CONFIG_INSTALLATION } from '@/types';
import { CenterThis, FadeLoader } from '@odigos/ui-kit/components';

export default function App() {
  const router = useRouter();
  const { config } = useConfig();

  useEffect(() => {
    if (config) {
      const { installation, readonly } = config;

      if (installation === CONFIG_INSTALLATION.NEW && !readonly) {
        router.push(ROUTES.CHOOSE_SOURCES);
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
