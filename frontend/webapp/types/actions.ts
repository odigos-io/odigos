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
