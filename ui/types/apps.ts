export enum AppKind {
  Deployment,
  StatefulSet,
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
