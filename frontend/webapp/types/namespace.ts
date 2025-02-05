export interface NamespaceFutureAppsInput {
  [namespace: string]: boolean;
}

export interface PersistNamespaceItemInput {
  name: string;
  futureSelected: boolean;
}
