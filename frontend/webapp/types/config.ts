import { InstallationMethod, OdigosConfig, Tier } from '@odigos/ui-kit/types';

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

export interface FetchedOdigosConfig extends Omit<OdigosConfig, 'nodeSelector' | 'userInstrumentationEnvs'> {
  nodeSelector: string; // JSON string representation of language mappings
}

export type OdigosConfigInput = Partial<FetchedOdigosConfig>;
