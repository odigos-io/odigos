'use client';
import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useNotify, useConfig } from '@/hooks';
import { FadeLoader } from '@/reuseable-components';
import { ROUTES, CONFIG, NOTIFICATION } from '@/utils';
import { CenterThis } from '@/styles';

export default function App() {
  const router = useRouter();
  const notify = useNotify();
  const { data, error } = useConfig();

  useEffect(() => {
    if (error) {
      notify({
        type: NOTIFICATION.ERROR,
        title: error.name,
        message: error.message,
      });
    } else if (data) {
      const { installation } = data;
      switch (installation) {
        // case CONFIG.FINISHED:
        // case CONFIG.APPS_SELECTED:
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
