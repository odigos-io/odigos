import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { useFilterStore } from '@/store/useFilterStore';
import { ACTION, deriveTypeFromRule, safeJsonParse } from '@/utils';
import { NOTIFICATION_TYPE, type ActionItem, type ComputePlatform, type ComputePlatformMapped } from '@/types';

type UseComputePlatformHook = {
  data?: ComputePlatformMapped;
  filteredData?: ComputePlatformMapped;
  loading: boolean;
  error?: Error;
  refetch: () => void;
};

export const useComputePlatform = (): UseComputePlatformHook => {
  const { addNotification } = useNotificationStore();

  // TODO: move filters to CRUD hooks
  const filters = useFilterStore();

  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  const mappedCP = useMemo(() => {
    if (!data) return undefined;

    return {
      computePlatform: {
        ...data.computePlatform,

        // sources are now paginated, refer to "usePaginatedSources" hook & "usePaginatedStore" store
        sources: undefined,

        actions: data.computePlatform.actions.map((item) => {
          const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ActionItem) : item.spec;

          return { ...item, spec: parsedSpec };
        }),

        instrumentationRules: data.computePlatform.instrumentationRules.map((item) => {
          const type = deriveTypeFromRule(item);

          return { ...item, type };
        }),

        destinations: data.computePlatform.destinations.map((item) => {
          // Replace deprecated string values, with boolean values
          const fields =
            item.destinationType.type === 'clickhouse'
              ? item.fields.replace('"CLICKHOUSE_CREATE_SCHEME":"Create"', '"CLICKHOUSE_CREATE_SCHEME":"true"').replace('"CLICKHOUSE_CREATE_SCHEME":"Skip"', '"CLICKHOUSE_CREATE_SCHEME":"false"')
              : item.destinationType.type === 'qryn'
              ? item.fields
                  .replace('"QRYN_ADD_EXPORTER_NAME":"Yes"', '"QRYN_ADD_EXPORTER_NAME":"true"')
                  .replace('"QRYN_ADD_EXPORTER_NAME":"No"', '"QRYN_ADD_EXPORTER_NAME":"false"')
                  .replace('"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"Yes"', '"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"true"')
                  .replace('"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"No"', '"QRYN_RESOURCE_TO_TELEMETRY_CONVERSION":"false"')
              : item.destinationType.type === 'qryn-oss'
              ? item.fields
                  .replace('"QRYN_OSS_ADD_EXPORTER_NAME":"Yes"', '"QRYN_OSS_ADD_EXPORTER_NAME":"true"')
                  .replace('"QRYN_OSS_ADD_EXPORTER_NAME":"No"', '"QRYN_OSS_ADD_EXPORTER_NAME":"false"')
                  .replace('"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"Yes"', '"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"true"')
                  .replace('"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"No"', '"QRYN_OSS_RESOURCE_TO_TELEMETRY_CONVERSION":"false"')
              : item.fields;

          return { ...item, fields };
        }),
      },
    };
  }, [data]);

  // TODO: move filters to CRUD hooks
  const filteredCP = useMemo(() => {
    if (!mappedCP) return undefined;

    let destinations = [...mappedCP.computePlatform.destinations];
    let actions = [...mappedCP.computePlatform.actions];

    if (!!filters.monitors.length) {
      destinations = destinations.filter((destination) => !!filters.monitors.find((metric) => destination.exportedSignals[metric.id]));
      actions = actions.filter((action) => !!filters.monitors.find((metric) => action.spec.signals.find((str) => str.toLowerCase() === metric.id)));
    }

    return {
      computePlatform: {
        ...mappedCP.computePlatform,
        destinations,
        actions,
      },
    };
  }, [mappedCP, filters]);

  return {
    data: mappedCP,
    filteredData: filteredCP,
    loading,
    error,
    refetch,
  };
};
