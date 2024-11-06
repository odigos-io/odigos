'use client';
import { useEffect } from 'react';
import { ROUTES, CONFIG } from '@/utils';
import { useRouter } from 'next/navigation';
import { useConfig, useNotify } from '@/hooks';
import { Loader } from '@keyval-dev/design-system';

export default function App() {
  const router = useRouter();
  const notify = useNotify();
  const { data, error } = useConfig();

  useEffect(() => {
    if (error) {
      notify({
        message: 'An error occurred',
        title: 'Error',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      });

      router.push(ROUTES.OVERVIEW);
    } else if (data) {
      const { installation } = data;

      switch (installation) {
        case CONFIG.NEW:
        case CONFIG.APPS_SELECTED:
          router.push(ROUTES.CHOOSE_SOURCES);
          break;
        case CONFIG.FINISHED:
          router.push(ROUTES.OVERVIEW);
      }
    }
  }, [data, error]);

  return <Loader />;
}
