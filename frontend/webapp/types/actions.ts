export enum ActionsType {
  ADD_CLUSTER_INFO = 'add-cluster-info',
}

export interface ActionItemCard {
  id: string;
  title: string;
  description: string;
  type: string;
  icon: string;
}

export interface ActionItem {
  id?: string;
  actionName: string;
  notes: string;
  signals: string[];
  [key: string]: any;
}
