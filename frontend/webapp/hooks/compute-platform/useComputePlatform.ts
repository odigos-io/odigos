import { useEffect, useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { useFilterStore } from '@/store/useFilterStore';
import { BACKEND_BOOLEAN, deriveTypeFromRule, safeJsonParse } from '@/utils';
import { NOTIFICATION_TYPE, type ActionItem, type ComputePlatform, type ComputePlatformMapped } from '@/types';

type UseComputePlatformHook = {
  data?: ComputePlatformMapped;
  filteredData?: ComputePlatformMapped;
  loading: boolean;
  error?: Error;
  refetch: () => void;
};

export const useComputePlatform = (): UseComputePlatformHook => {
  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM);
  const { addNotification } = useNotificationStore();
  const filters = useFilterStore();

  useEffect(() => {
    if (error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name,
        message: error.cause?.message,
      });
    }
  }, [error]);

  const mappedData = useMemo(() => {
    if (!data) return undefined;

    return {
      computePlatform: {
        ...data.computePlatform,
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

  const filteredData = useMemo(() => {
    if (!mappedData) return undefined;

    let k8sActualSources = [...mappedData.computePlatform.k8sActualSources];
    let destinations = [...mappedData.computePlatform.destinations];
    let actions = [...mappedData.computePlatform.actions];

    if (!!filters.namespace) {
      k8sActualSources = k8sActualSources.filter((source) => filters.namespace?.id === source.namespace);
    }
    if (!!filters.types.length) {
      k8sActualSources = k8sActualSources.filter((source) => !!filters.types.find((type) => type.id === source.kind));
    }
    if (!!filters.onlyErrors) {
      k8sActualSources = k8sActualSources.filter((source) => !!source.conditions?.find((cond) => cond.status === BACKEND_BOOLEAN.FALSE));
    }
    if (!!filters.errors.length) {
      k8sActualSources = k8sActualSources.filter((source) => !!filters.errors.find((error) => !!source.conditions?.find((cond) => cond.message === error.id)));
    }
    if (!!filters.languages.length) {
      k8sActualSources = k8sActualSources.filter((source) => !!filters.languages.find((language) => !!source.containers?.find((cont) => cont.language === language.id)));
    }
    if (!!filters.monitors.length) {
      destinations = destinations.filter((destination) => !!filters.monitors.find((metric) => destination.exportedSignals[metric.id]));
      actions = actions.filter((action) => !!filters.monitors.find((metric) => action.spec.signals.find((str) => str.toLowerCase() === metric.id)));
    }

    return {
      computePlatform: {
        ...mappedData.computePlatform,
        k8sActualSources,
        destinations,
        actions,
      },
    };
  }, [mappedData, filters]);

  return { data: mappedData, filteredData, loading, error, refetch };
};
