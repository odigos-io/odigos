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
  type: string;
  [key: string]: any;
}

export interface ActionData {
  id: string;
  type: string;
  spec: ActionItem | string;
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
