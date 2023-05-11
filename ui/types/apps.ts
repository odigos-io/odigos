export enum AppKind {
  Deployment,
  StatefulSet,
  DaemonSet,
}

export type ApplicationData = {
  id: string;
  name: string;
  languages: string[];
  instrumented: boolean;
  kind: AppKind;
  namespace: string;
};

export type AppsApiResponse = {
  apps: ApplicationData[];
  discovery_in_progress: boolean;
};


export type KubernetesObject = {
  name: string;
  kind: AppKind;
  instances: number;
  labeled: boolean;
};

export type KubernetesNamespace = {
  name: string;
  labeled: boolean;
  objects: KubernetesObject[];
};

export type KubernetesObjectsInNamespaces = {
  namespaces: KubernetesNamespace[];
};