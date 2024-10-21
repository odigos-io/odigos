'use client';
import { useEffect } from 'react';
import { useConfig } from '@/hooks';
import { ROUTES, CONFIG } from '@/utils';
import { useRouter } from 'next/navigation';
import { addNotification, store } from '@/store';
import { Loader } from '@keyval-dev/design-system';

export default function App() {
  const router = useRouter();
  const { data, error } = useConfig();

  useEffect(() => {
    data && renderCurrentPage();
  }, [data, error]);

  useEffect(() => {
    if (!error) return;
    store.dispatch(
      addNotification({
        id: '1',
        message: 'An error occurred',
        title: 'Error',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      })
    );
    router.push(ROUTES.OVERVIEW);
  }, [error]);

  function renderCurrentPage() {
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

  return <Loader />;
}
