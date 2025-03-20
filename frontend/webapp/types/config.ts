import { Tier } from '@odigos/ui-kit/types';

export enum ConfigInstallation {
  New = 'NEW',
  AppsSelected = 'APPS_SELECTED',
  Finished = 'FINISHED',
}

export interface FetchedConfig {
  installation: ConfigInstallation;
  tier: Tier;
  readonly: boolean;
}
