'use client';

import { useEffect } from 'react';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { useOdigosApi } from '@odigos/ui-kit/contexts/odigos-api';
import { Crud, InstallationStatus, StatusType, Tier } from '@odigos/ui-kit/types';

/**
 * Reads the cluster's `FetchedConfig` via the kit's typed `GET_CONFIG`
 * slot. Apollo handles the fetch + cache; the kit's slot is bare-typed
 * so consumers don't have to narrow.
 *
 * Returns the same shape as before (`config` / `isReadonly` /
 * `isEnterprise` / `installationStatus`) so existing call sites in the
 * webapp's layouts and pages don't need to change.
 */
export const useConfig = () => {
  const { addNotification } = useNotificationStore();
  const { data: config, error } = useOdigosApi().configApi.useEffectiveConfig();

  useEffect(() => {
    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause instanceof Error ? error.cause.message : error.message,
      });
    }
  }, [error, addNotification]);

  const isReadonly = config?.readonly || false;
  const isEnterprise = (config?.tier && [Tier.Onprem].includes(config.tier)) || false;
  const installationStatus = config?.installationStatus || InstallationStatus.New;

  return { config, isReadonly, isEnterprise, installationStatus };
};
