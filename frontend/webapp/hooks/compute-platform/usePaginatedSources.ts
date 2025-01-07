import { useEffect } from 'react';
import { ACTION } from '@/utils';
import { GET_SOURCES } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore, usePaginatedStore } from '@/store';
import { NOTIFICATION_TYPE, type ComputePlatform } from '@/types';

export const usePaginatedSources = () => {
  const { addNotification } = useNotificationStore();
  const { sources, addSources, setSources } = usePaginatedStore();

  const [getSources, { loading }] = useLazyQuery<{ computePlatform: { sources: ComputePlatform['computePlatform']['sources'] } }>(GET_SOURCES, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  const fetchSources = async (getAll: boolean = false, nextPage: string = '') => {
    if (nextPage === '') setSources([]);
    const { data } = await getSources({ variables: { nextPage } });

    if (getAll && !!data?.computePlatform.sources) {
      const { nextPage, items } = data.computePlatform.sources;

      addSources(items);
      // This timeout is to prevent react-flow from flickering on re-renders
      setTimeout(() => !!nextPage && fetchSources(true, nextPage), 100);
    }
  };

  // TODO: paginate all on a button click
  useEffect(() => {
    if (!sources.length && !loading) fetchSources(true);
  }, []);

  return {
    sources,
    fetchSources,
  };
};
