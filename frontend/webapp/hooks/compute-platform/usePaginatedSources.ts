import { useEffect } from 'react';
import { ACTION } from '@/utils';
import { GET_SOURCES } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore, usePaginatedStore } from '@/store';
import { NOTIFICATION_TYPE, type ComputePlatform } from '@/types';

export const usePaginatedSources = () => {
  const { addNotification } = useNotificationStore();
  const { sources, addSources, setSources, sourcesNotFinished, setSourcesNotFinished } = usePaginatedStore();

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

    if (!!data?.computePlatform.sources) {
      const { nextPage, items } = data.computePlatform.sources;

      addSources(items);

      if (getAll) {
        // This timeout is to prevent react-flow from flickering on re-renders
        setTimeout(() => {
          if (!!nextPage) fetchSources(true, nextPage);
          else setSourcesNotFinished(false);
        }, 10);
      } else if (!!nextPage) {
        setSourcesNotFinished(true);
      }
    }
  };

  // Fetch 1 batch on initial mount
  useEffect(() => {
    if (!sources.length && !loading) fetchSources();
  }, []);

  return {
    sources,
    fetchSources,
    sourcesNotFinished,
  };
};
