import { GET_METRICS } from '@/graphql';
import { useQuery } from '@apollo/client';
import type { Metrics } from '@odigos/ui-kit/types';
import { useDrawerStore, useEntityStore, useModalStore } from '@odigos/ui-kit/store';

export const useMetrics = () => {
  const { drawerType } = useDrawerStore();
  const { currentModal } = useModalStore();
  const sources = useEntityStore((state) => state.sources);
  const destinations = useEntityStore((state) => state.destinations);

  const { data } = useQuery<{ getOverviewMetrics: Metrics }>(GET_METRICS, {
    skip: !sources.length || !destinations.length || !!drawerType || !!currentModal,
    pollInterval: 10000,
  });

  return {
    metrics: data?.getOverviewMetrics || {
      sources: [],
      destinations: [],
    },
  };
};
