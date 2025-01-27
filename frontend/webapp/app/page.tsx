'use client';
import { useEffect } from 'react';
import { useConfig } from '@/hooks';
import { CenterThis } from '@/styles';
import { ROUTES, CONFIG } from '@/utils';
import { useRouter } from 'next/navigation';
import { FadeLoader } from '@/reuseable-components';

export default function App() {
  const router = useRouter();
  const { data } = useConfig();

  useEffect(() => {
    if (data) {
      const { installation, readonly } = data;

      if (installation === CONFIG.NEW && !readonly) {
        router.push(ROUTES.CHOOSE_SOURCES);
      } else {
        router.push(ROUTES.OVERVIEW);
      }
    }
  }, [data]);

  return (
    <CenterThis style={{ height: '100%' }}>
      <FadeLoader style={{ scale: 2 }} />
    </CenterThis>
  );
}
