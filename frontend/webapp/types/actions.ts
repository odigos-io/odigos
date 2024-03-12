export enum ActionsType {
  ADD_CLUSTER_INFO = 'AddClusterInfo',
  DELETE_ATTRIBUTES = 'deleteattribute',
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
  [key: string]: any;
}

export interface ActionData {
  id: string;
  type: string;
  spec: ActionItem;
}
