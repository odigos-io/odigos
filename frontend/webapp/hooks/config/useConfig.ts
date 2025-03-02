'use client';
import { useEffect } from 'react';
import { GET_CONFIG } from '@/graphql';
import { type FetchedConfig } from '@/@types';
import { useSuspenseQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-containers';
import { CRUD, NOTIFICATION_TYPE, TIER } from '@odigos/ui-utils';

export const useConfig = () => {
  const { addNotification } = useNotificationStore();

  const { data, error } = useSuspenseQuery<FetchedConfig>(GET_CONFIG, {
    skip: typeof window === 'undefined',
  });

  useEffect(() => {
    if (error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  const cfg = data?.config;
  const isCommunity = !!cfg?.tier && [TIER.COMMUNITY].includes(cfg.tier);
  const isEnterprise = !!cfg?.tier && [TIER.ONPREM].includes(cfg.tier);

  return { data: cfg, isCommunity, isEnterprise };
};
