'use client';
import { useEffect } from 'react';
import { useQuery } from 'react-query';
import { useRouter } from 'next/navigation';
import { ROUTES, CONFIG, QUERIES } from '@/utils';
import { Loader } from '@keyval-dev/design-system';
import { getDestinations, getConfig } from '@/services';
import { addNotification, store } from '@/store';
export default function App() {
  const router = useRouter();
  const { data, isLoading: isConfigLoading } = useQuery(
    [QUERIES.API_CONFIG],
    getConfig
  );
  const {
    isLoading: isDestinationLoading,
    data: destinationList,
    error,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  useEffect(() => {
    if (isConfigLoading || isDestinationLoading || error) return;

    renderCurrentPage();
  }, [data, destinationList, isConfigLoading || error]);

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

    if (destinationList.length > 0) {
      router.push(ROUTES.OVERVIEW);
      return;
    }

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
