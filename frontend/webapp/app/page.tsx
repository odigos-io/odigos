'use client';
import { useEffect } from 'react';
import { useQuery } from 'react-query';
import { useRouter } from 'next/navigation';
import { getDestinations } from '@/services';
import { getConfig } from '@/services/config';
import { Loader } from '@keyval-dev/design-system';
import { ROUTES, CONFIG, QUERIES } from '@/utils/constants';

export default function App() {
  const router = useRouter();
  const { data, isLoading: isConfigLoading } = useQuery(
    [QUERIES.API_CONFIG],
    getConfig
  );
  const { isLoading: isDestinationLoading, data: destinationList } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );
  useEffect(() => {
    if (isConfigLoading || isDestinationLoading) return;

    renderCurrentPage();
  }, [data, destinationList]);

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
