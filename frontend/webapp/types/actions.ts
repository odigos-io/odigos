import type { Condition } from './common';
import { ACTION_TYPE, SIGNAL_TYPE } from '@odigos/ui-utils';

export type AddClusterInfoSpec = {
  clusterAttributes: {
    attributeName: string;
    attributeStringValue: string;
  }[];
};

export type DeleteAttributesSpec = {
  attributeNamesToDelete: string[];
};

export type RenameAttributesSpec = {
  renames: {
    [oldKey: string]: string;
  };
};

export type PiiMaskingSpec = {
  piiCategories: string[];
};

export type ErrorSamplerSpec = {
  fallback_sampling_ratio: number;
};

export type ProbabilisticSamplerSpec = {
  sampling_percentage: string;
};

export type LatencySamplerSpec = {
  endpoints_filters: {
    service_name: string;
    http_route: string;
    minimum_latency_threshold: number;
    fallback_sampling_ratio: number;
  }[];
};

export interface ActionItem {
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

export interface ActionData {
  id: string;
  type: ACTION_TYPE;
  spec: ActionItem | string;
  conditions: Condition[] | null;
}

export interface ActionDataParsed extends ActionData {
  spec: ActionItem;
}

export type ActionInput = {
  type: ACTION_TYPE;
  name: string;
  notes: string;
  disable: boolean;
  signals: SIGNAL_TYPE[];
  details: string;
};
