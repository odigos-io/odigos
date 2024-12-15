'use client';
import { useEffect } from 'react';
import { CenterThis } from '@/styles';
import { ROUTES, CONFIG } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import { useRouter } from 'next/navigation';
import { useNotify, useConfig } from '@/hooks';
import { FadeLoader } from '@/reuseable-components';

export default function App() {
  const router = useRouter();
  const notify = useNotify();
  const { data, error } = useConfig();

  useEffect(() => {
    if (error) {
      notify({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name,
        message: error.message,
      });
    } else if (data) {
      const { installation } = data;
      switch (installation) {
        case CONFIG.NEW:
          router.push(ROUTES.CHOOSE_SOURCES);
          break;
        default:
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
