import { InstallationMethod, Tier } from '@odigos/ui-kit/types';

export enum InstallationStatus {
  New = 'NEW',
  AppsSelected = 'APPS_SELECTED',
  Finished = 'FINISHED',
}

export interface FetchedConfig {
  readonly: boolean;
  tier: Tier;
  installationMethod: InstallationMethod;
  installationStatus: InstallationStatus;
}
