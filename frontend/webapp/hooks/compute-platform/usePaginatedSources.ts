import { useEffect } from 'react';
import { ACTION } from '@/utils';
import { GET_SOURCES } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { useNotificationStore, usePaginatedStore } from '@/store';
import { NOTIFICATION_TYPE, type ComputePlatform } from '@/types';

export const usePaginatedSources = () => {
  const { addNotification } = useNotificationStore();
  const { sources, addSources, setSources, sourcesNotFinished, setSourcesNotFinished, sourcesFetching, setSourcesFetching } = usePaginatedStore();

  const [getSources, { loading }] = useLazyQuery<{ computePlatform: { sources: ComputePlatform['computePlatform']['sources'] } }>(GET_SOURCES, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  const fetchSources = async (getAll: boolean = true, nextPage: string = '') => {
    if (nextPage === '') setSources([]);
    setSourcesFetching(true);
    const { data } = await getSources({ variables: { nextPage } });

    if (!!data?.computePlatform?.sources) {
      const { nextPage, items } = data.computePlatform.sources;

      addSources(items);

      if (getAll) {
        if (!!nextPage) {
          // This timeout is to prevent react-flow from flickering on re-renders
          setTimeout(() => fetchSources(true, nextPage), 10);
        } else {
          setSourcesNotFinished(false);
          setSourcesFetching(false);
        }
      } else if (!!nextPage) {
        setSourcesNotFinished(true);
        setSourcesFetching(false);
      }
    }
  };

  // Fetch 1 batch on initial mount
  useEffect(() => {
    if (!sources.length && !loading && !sourcesFetching) fetchSources();
  }, []);

  return {
    sources,
    fetchSources,
    sourcesNotFinished,
    sourcesFetching,
  };
};
