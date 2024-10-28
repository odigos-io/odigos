import { type SignalUppercase } from '@/utils';

export enum ActionsType {
  ADD_CLUSTER_INFO = 'AddClusterInfo',
  DELETE_ATTRIBUTES = 'DeleteAttribute',
  RENAME_ATTRIBUTES = 'RenameAttribute',
  ERROR_SAMPLER = 'ErrorSampler',
  PROBABILISTIC_SAMPLER = 'ProbabilisticSampler',
  LATENCY_SAMPLER = 'LatencySampler',
  PII_MASKING = 'PiiMasking',
}

export enum ActionsSortType {
  ACTION_NAME = 'action_name',
  STATUS = 'status',
  TYPE = 'type',
}

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

export interface ActionItemCard {
  id: string;
  title: string;
  description: string;
  type: string;
  icon: string;
}

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
  type: string;
  spec: ActionItem | string;
}

export interface ActionDataParsed extends ActionData {
  spec: ActionItem;
}

interface Monitor {
  id: string;
  label: string;
  checked: boolean;
}

export interface ActionState {
  id?: string;
  actionName: string;
  actionNote: string;
  actionData: any;
  selectedMonitors: Monitor[];
  disabled: boolean;
  type: string;
}

export type ActionInput = {
  type: string;
  name: string;
  notes: string;
  disable: boolean;
  signals: SignalUppercase[];
  details: string;
};
