'use client';

import { useEffect } from 'react';
import { GET_CONFIG } from '@/graphql';
import type { FetchedConfig } from '@/types';
import { useSuspenseQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, Tier } from '@odigos/ui-kit/types';

export const useConfig = () => {
  const { addNotification } = useNotificationStore();

  const { data, error } = useSuspenseQuery<{ config?: FetchedConfig }>(GET_CONFIG, {
    skip: typeof window === 'undefined',
  });

  useEffect(() => {
    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  const config = data?.config;
  const isReadonly = data?.config?.readonly || false;
  const isCommunity = (config?.tier && [Tier.Community].includes(config.tier)) || false;
  const isEnterprise = (config?.tier && [Tier.Onprem].includes(config.tier)) || false;

  return { config, isReadonly, isCommunity, isEnterprise };
};
