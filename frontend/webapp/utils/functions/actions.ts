import { safeJsonParse } from '@odigos/ui-kit/functions';
import type { ActionInput, FetchedAction, ParsedActionSpec } from '@/types';
import { ACTION_TYPE, SIGNAL_TYPE, type Action, type ActionFormData } from '@odigos/ui-kit/types';

export const mapFetchedActions = (items: FetchedAction[]): Action[] => {
  return items.map((item) => {
    const type = item.type;
    const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ParsedActionSpec) : item.spec;
    const spec: Partial<Action['spec']> = {};

    switch (type) {
      case ACTION_TYPE.K8S_ATTRIBUTES:
        spec.collectContainerAttributes = parsedSpec.collectContainerAttributes || false;
        spec.collectWorkloadId = parsedSpec.collectWorkloadUID || false;
        spec.collectClusterId = parsedSpec.collectClusterUID || false;
        spec.labelsAttributes = parsedSpec.labelsAttributes;
        spec.annotationsAttributes = parsedSpec.annotationsAttributes;
        break;

      case ACTION_TYPE.ADD_CLUSTER_INFO:
        spec.clusterAttributes = parsedSpec.clusterAttributes;
        break;

      case ACTION_TYPE.DELETE_ATTRIBUTES:
        spec.attributeNamesToDelete = parsedSpec.attributeNamesToDelete;
        break;

      case ACTION_TYPE.RENAME_ATTRIBUTES:
        spec.renames = parsedSpec.renames;
        break;

      case ACTION_TYPE.PII_MASKING:
        spec.piiCategories = parsedSpec.piiCategories;
        break;

      case ACTION_TYPE.ERROR_SAMPLER:
        spec.fallbackSamplingRatio = parsedSpec.fallback_sampling_ratio;
        break;

      case ACTION_TYPE.PROBABILISTIC_SAMPLER:
        spec.samplingPercentage = Number(parsedSpec.sampling_percentage);
        break;

      case ACTION_TYPE.LATENCY_SAMPLER:
        spec.endpointsFilters = parsedSpec.endpoints_filters?.map(({ service_name, http_route, minimum_latency_threshold, fallback_sampling_ratio }) => ({
          serviceName: service_name,
          httpRoute: http_route,
          minimumLatencyThreshold: minimum_latency_threshold,
          fallbackSamplingRatio: fallback_sampling_ratio,
        }));
        break;

      default:
        break;
    }

    return {
      ...item,
      spec: {
        ...spec,
        actionName: parsedSpec.actionName,
        notes: parsedSpec.notes,
        disabled: parsedSpec.disabled,
        signals: parsedSpec.signals.map((str) => str.toLowerCase() as SIGNAL_TYPE),
      },
    };
  });
};

export const mapActionsFormToGqlInput = (action: ActionFormData): ActionInput => {
  const {
    type,
    name = '',
    notes = '',
    disabled = false,
    signals,
    collectContainerAttributes,
    collectWorkloadId,
    collectClusterId,
    labelsAttributes,
    annotationsAttributes,
    clusterAttributes,
    attributeNamesToDelete,
    renames,
    piiCategories,
    fallbackSamplingRatio,
    samplingPercentage,
    endpointsFilters,
  } = action;

  const payload: ActionInput = {
    type,
    name,
    notes,
    disable: disabled,
    signals: signals.map((signal) => signal.toUpperCase()),
    details: '',
  };

  switch (type) {
    case ACTION_TYPE.K8S_ATTRIBUTES:
      payload['details'] = JSON.stringify({
        collectContainerAttributes,
        collectWorkloadId,
        collectClusterId,
        labelsAttributes,
        annotationsAttributes,
      });
      break;

    case ACTION_TYPE.ADD_CLUSTER_INFO:
      payload['details'] = JSON.stringify({ clusterAttributes });
      break;

    case ACTION_TYPE.DELETE_ATTRIBUTES:
      payload['details'] = JSON.stringify({ attributeNamesToDelete });
      break;

    case ACTION_TYPE.RENAME_ATTRIBUTES:
      payload['details'] = JSON.stringify({ renames });
      break;

    case ACTION_TYPE.PII_MASKING:
      payload['details'] = JSON.stringify({ piiCategories });
      break;

    case ACTION_TYPE.ERROR_SAMPLER:
      payload['details'] = JSON.stringify({ fallback_sampling_ratio: fallbackSamplingRatio });
      break;

    case ACTION_TYPE.PROBABILISTIC_SAMPLER:
      payload['details'] = JSON.stringify({ sampling_percentage: String(samplingPercentage) });
      break;

    case ACTION_TYPE.LATENCY_SAMPLER:
      payload['details'] = JSON.stringify({
        endpoints_filters:
          endpointsFilters?.map(({ serviceName, httpRoute, minimumLatencyThreshold, fallbackSamplingRatio }) => ({
            service_name: serviceName,
            http_route: httpRoute,
            minimum_latency_threshold: minimumLatencyThreshold,
            fallback_sampling_ratio: fallbackSamplingRatio,
          })) || [],
      });
      break;

    default:
      break;
  }

  return payload;
};
