'use client';

import { useApiQuery } from '@odigos/ui-kit/contexts/odigos-api';
import { FetchedConfig, InstallationStatus, Tier } from '@odigos/ui-kit/types';

export const useConfig = (): { config: FetchedConfig; isReadonly: boolean; isEnterprise: boolean; installationStatus: InstallationStatus } => {
  const { data: config } = useApiQuery('GET_CONFIG');

  const isReadonly = config?.readonly || false;
  const isEnterprise = (config?.tier && [Tier.Onprem].includes(config.tier)) || false;
  const installationStatus = config?.installationStatus || InstallationStatus.New;

  return { config, isReadonly, isEnterprise, installationStatus };
};
