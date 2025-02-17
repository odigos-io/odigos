'use client';

import { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useConfig } from '@/hooks';
import { useRouter } from 'next/navigation';
import { CONFIG_INSTALLATION } from '@/@types';
import { CenterThis, FadeLoader } from '@odigos/ui-components';
import PageContainer from '@/components/providers/page-container';

export default function App() {
  const router = useRouter();
  const { data } = useConfig();

  useEffect(() => {
    if (data) {
      const { installation, readonly } = data;

      if (installation === CONFIG_INSTALLATION.NEW && !readonly) {
        router.push(ROUTES.CHOOSE_SOURCES);
      } else {
        router.push(ROUTES.OVERVIEW);
      }
    }
  }, [data]);

  return (
    <PageContainer>
      <CenterThis style={{ height: '100%' }}>
        <FadeLoader scale={2} />
      </CenterThis>
    </PageContainer>
  );
}
