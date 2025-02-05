import { ACTION_TYPE, type FetchedCondition, SIGNAL_TYPE } from '@odigos/ui-utils';

export interface AddClusterInfoSpec {
  clusterAttributes: {
    attributeName: string;
    attributeStringValue: string;
  }[];
}

export interface DeleteAttributesSpec {
  attributeNamesToDelete: string[];
}

export interface RenameAttributesSpec {
  renames: {
    [oldKey: string]: string;
  };
}

export interface PiiMaskingSpec {
  piiCategories: string[];
}

export interface ErrorSamplerSpec {
  fallback_sampling_ratio: number;
}

export interface ProbabilisticSamplerSpec {
  sampling_percentage: string;
}

export interface LatencySamplerSpec {
  endpoints_filters: {
    service_name: string;
    http_route: string;
    minimum_latency_threshold: number;
    fallback_sampling_ratio: number;
  }[];
}

export interface FetchedActionSpec {
  actionName: string;
  notes: string;
  signals: string[];
  disabled?: boolean;
  clusterAttributes?: AddClusterInfoSpec['clusterAttributes'];
  attributeNamesToDelete?: DeleteAttributesSpec['attributeNamesToDelete'];
  renames?: RenameAttributesSpec['renames'];
  piiCategories?: PiiMaskingSpec['piiCategories'];
  fallback_sampling_ratio?: ErrorSamplerSpec['fallback_sampling_ratio'];
  sampling_percentage?: ProbabilisticSamplerSpec['sampling_percentage'];
  endpoints_filters?: LatencySamplerSpec['endpoints_filters'];
}

export interface FetchedAction {
  id: string;
  type: ACTION_TYPE;
  spec: string;
  conditions: FetchedCondition[];
}

export interface ActionInput {
  type: ACTION_TYPE;
  name: string;
  notes: string;
  disable: boolean;
  signals: SIGNAL_TYPE[];
  details: string;
}
