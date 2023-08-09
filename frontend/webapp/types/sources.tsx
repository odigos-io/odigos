export interface ManagedSource {
  kind: string;
  name: string;
  namespace: string;
  reported_name?: string;
  languages: [
    {
      container_name: string;
      language: string;
    }
  ];
}

export interface Namespace {
  name: string;
  selected: boolean;
  totalApps: number;
}

export interface SelectedSources {
  [key: string]: {
    objects: {
      name: string;
      selected: boolean;
      kind: string;
      app_instrumentation_labeled: boolean | null;
      ns_instrumentation_labeled: boolean | null;
      instrumentation_effective: boolean | null;
      instances: number;
    };
    selected_all: boolean;
    future_selected: boolean;
  };
}
