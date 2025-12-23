import { InstallationMethod, PlatformType, Tier } from '@odigos/ui-kit/types';

export enum InstallationStatus {
  New = 'NEW',
  AppsSelected = 'APPS_SELECTED',
  Finished = 'FINISHED',
}

export interface FetchedConfig {
  readonly: boolean;
  platformType: PlatformType;
  tier: Tier;
  odigosVersion: string;
  installationMethod: InstallationMethod;
  installationStatus: InstallationStatus;
}
