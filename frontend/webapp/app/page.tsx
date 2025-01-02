'use client';
import { useEffect } from 'react';
import { useConfig } from '@/hooks';
import { CenterThis } from '@/styles';
import { NOTIFICATION_TYPE } from '@/types';
import { useRouter } from 'next/navigation';
import { useNotificationStore } from '@/store';
import { ROUTES, CONFIG, ACTION } from '@/utils';
import { FadeLoader } from '@/reuseable-components';

export default function App() {
  const router = useRouter();
  const { data, error } = useConfig();
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    if (error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
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
