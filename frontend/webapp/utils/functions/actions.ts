import { safeJsonParse } from '@odigos/ui-kit/functions';
import type { ActionInput, FetchedAction, ParsedActionSpec } from '@/types';
import { ActionType, SignalType, type Action, type ActionFormData } from '@odigos/ui-kit/types';

export const mapFetchedActions = (items: FetchedAction[]): Action[] => {
  return items.map((item) => {
    const type = item.type;
    const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ParsedActionSpec) : item.spec;
    const spec: Partial<Action['spec']> = {};

    switch (type) {
      case ActionType.K8sAttributes:
        spec.collectContainerAttributes = parsedSpec.collectContainerAttributes || false;
        spec.collectReplicaSetAttributes = parsedSpec.collectReplicaSetAttributes || false;
        spec.collectWorkloadId = parsedSpec.collectWorkloadUID || false;
        spec.collectClusterId = parsedSpec.collectClusterUID || false;
        spec.labelsAttributes = parsedSpec.labelsAttributes;
        spec.annotationsAttributes = parsedSpec.annotationsAttributes;
        break;

      case ActionType.AddClusterInfo:
        spec.clusterAttributes = parsedSpec.clusterAttributes;
        break;

      case ActionType.DeleteAttributes:
        spec.attributeNamesToDelete = parsedSpec.attributeNamesToDelete;
        break;

      case ActionType.RenameAttributes:
        spec.renames = parsedSpec.renames;
        break;

      case ActionType.PiiMasking:
        spec.piiCategories = parsedSpec.piiCategories;
        break;

      case ActionType.ErrorSampler:
        spec.fallbackSamplingRatio = parsedSpec.fallback_sampling_ratio;
        break;

      case ActionType.ProbabilisticSampler:
        spec.samplingPercentage = Number(parsedSpec.sampling_percentage);
        break;

      case ActionType.LatencySampler:
        spec.endpointsFilters = parsedSpec.endpoints_filters?.map(({ service_name, http_route, minimum_latency_threshold, fallback_sampling_ratio }) => ({
          serviceName: service_name,
          httpRoute: http_route,
          minimumLatencyThreshold: minimum_latency_threshold,
          fallbackSamplingRatio: fallback_sampling_ratio,
        }));
        break;

      case ActionType.ServiceNameSampler:
        spec.servicesNameFilters = parsedSpec.services_name_filters?.map(({ service_name, sampling_ratio, fallback_sampling_ratio }) => ({
          serviceName: service_name,
          samplingRatio: sampling_ratio,
          fallbackSamplingRatio: fallback_sampling_ratio,
        }));
        break;

      case ActionType.SpanAttributeSampler:
        spec.attributeFilters = parsedSpec.attribute_filters?.map(({ service_name, attribute_key, fallback_sampling_ratio, condition }) => ({
          serviceName: service_name,
          attributeKey: attribute_key,
          fallbackSamplingRatio: fallback_sampling_ratio,
          condition,
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
        signals: parsedSpec.signals.map((str) => str.toLowerCase() as SignalType),
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
    collectReplicaSetAttributes,
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
    servicesNameFilters,
    attributeFilters,
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
    case ActionType.K8sAttributes:
      payload['details'] = JSON.stringify({
        collectContainerAttributes,
        collectReplicaSetAttributes,
        collectWorkloadId,
        collectClusterId,
        labelsAttributes,
        annotationsAttributes,
      });
      break;

    case ActionType.AddClusterInfo:
      payload['details'] = JSON.stringify({ clusterAttributes });
      break;

    case ActionType.DeleteAttributes:
      payload['details'] = JSON.stringify({ attributeNamesToDelete });
      break;

    case ActionType.RenameAttributes:
      payload['details'] = JSON.stringify({ renames });
      break;

    case ActionType.PiiMasking:
      payload['details'] = JSON.stringify({ piiCategories });
      break;

    case ActionType.ErrorSampler:
      payload['details'] = JSON.stringify({ fallback_sampling_ratio: fallbackSamplingRatio });
      break;

    case ActionType.ProbabilisticSampler:
      payload['details'] = JSON.stringify({ sampling_percentage: String(samplingPercentage) });
      break;

    case ActionType.LatencySampler:
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

    case ActionType.ServiceNameSampler:
      payload['details'] = JSON.stringify({
        services_name_filters:
          servicesNameFilters?.map(({ serviceName, samplingRatio, fallbackSamplingRatio }) => ({
            service_name: serviceName,
            sampling_ratio: samplingRatio,
            fallback_sampling_ratio: fallbackSamplingRatio,
          })) || [],
      });
      break;

    case ActionType.SpanAttributeSampler:
      payload['details'] = JSON.stringify({
        attribute_filters: attributeFilters?.map(({ serviceName, attributeKey, fallbackSamplingRatio, condition }) => ({
          service_name: serviceName,
          attribute_key: attributeKey,
          fallback_sampling_ratio: fallbackSamplingRatio,
          condition,
        })),
      });
      break;

    default:
      break;
  }

  return payload;
};
