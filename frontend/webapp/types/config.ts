import { TIER } from '@odigos/ui-kit/types';

export enum CONFIG_INSTALLATION {
  NEW = 'NEW',
  APPS_SELECTED = 'APPS_SELECTED',
  FINISHED = 'FINISHED',
}

export interface FetchedConfig {
  installation: CONFIG_INSTALLATION;
  tier: TIER;
  readonly: boolean;
}
