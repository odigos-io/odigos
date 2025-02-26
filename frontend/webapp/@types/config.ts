import { TIER } from '@odigos/ui-utils';

export enum CONFIG_INSTALLATION {
  NEW = 'NEW',
  APPS_SELECTED = 'APPS_SELECTED',
  FINISHED = 'FINISHED',
}

export interface FetchedConfig {
  config: {
    installation: CONFIG_INSTALLATION;
    tier: TIER;
    readonly: boolean;
  };
}
