export enum ActionsType {
  INSERT_CLUSTER_ATTRIBUTES = 'insert-cluster-attributes',
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
