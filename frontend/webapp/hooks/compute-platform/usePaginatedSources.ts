import { useEffect } from 'react';
import { GET_SOURCES } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { ACTION, WORKLOAD_PROGRAMMING_LANGUAGES } from '@/utils';
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
    // const { data } = await getSources({ variables: { nextPage } });

    const data = {
      computePlatform: {
        sources: {
          nextPage: '',
          items: [
            {
              namespace: 'default',
              name: 'coupon',
              kind: 'Deployment',
              selected: true,
              reportedName: 'coupon',
              containers: [
                {
                  containerName: 'coupon',
                  language: 'javascript' as WORKLOAD_PROGRAMMING_LANGUAGES,
                  runtimeVersion: '18.3.0',
                  otherAgent: null,
                },
              ],
              conditions: [
                {
                  status: 'False',
                  type: 'AppliedInstrumentationDevice',
                  reason: 'DataCollectionNotReady',
                  message: 'OpenTelemetry pipeline not yet ready to receive data',
                  lastTransitionTime: '2025-01-21T12:04:01Z',
                },
              ],
            },
            {
              namespace: 'default',
              name: 'frontend',
              kind: 'Deployment',
              selected: true,
              reportedName: 'frontend',
              containers: [
                {
                  containerName: 'frontend',
                  language: 'java' as WORKLOAD_PROGRAMMING_LANGUAGES,
                  runtimeVersion: '17.0.12+7',
                  otherAgent: null,
                },
              ],
              conditions: [],
            },
            {
              namespace: 'default',
              name: 'inventory',
              kind: 'Deployment',
              selected: true,
              reportedName: 'inventory',
              containers: [
                {
                  containerName: 'inventory',
                  language: 'python' as WORKLOAD_PROGRAMMING_LANGUAGES,
                  runtimeVersion: '3.11.9',
                  otherAgent: null,
                },
              ],
              conditions: [
                {
                  status: 'False',
                  type: 'AppliedInstrumentationDevice',
                  reason: 'DataCollectionNotReady',
                  message: 'OpenTelemetry pipeline not yet ready to receive data',
                  lastTransitionTime: '2025-01-21T12:04:00Z',
                },
              ],
            },
            {
              namespace: 'default',
              name: 'membership',
              kind: 'Deployment',
              selected: true,
              reportedName: 'membership',
              containers: [
                {
                  containerName: 'membership',
                  language: 'go' as WORKLOAD_PROGRAMMING_LANGUAGES,
                  runtimeVersion: '1.21.4',
                  otherAgent: null,
                },
              ],
              conditions: [],
            },
            {
              namespace: 'default',
              name: 'pricing',
              kind: 'Deployment',
              selected: true,
              reportedName: 'pricing',
              containers: [
                {
                  containerName: 'pricing',
                  language: 'dotnet' as WORKLOAD_PROGRAMMING_LANGUAGES,
                  runtimeVersion: '',
                  otherAgent: null,
                },
              ],
              conditions: [
                {
                  status: 'False',
                  type: 'AppliedInstrumentationDevice',
                  reason: 'DataCollectionNotReady',
                  message: 'OpenTelemetry pipeline not yet ready to receive data',
                  lastTransitionTime: '2025-01-21T12:04:01Z',
                },
              ],
            },
          ],
        },
      },
    };

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
