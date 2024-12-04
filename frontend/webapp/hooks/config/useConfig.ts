'use client';
import { useEffect } from 'react';
import { Config } from '@/types';
import { GET_CONFIG } from '@/graphql';
import { useSuspenseQuery } from '@apollo/client';

export const useConfig = () => {
  const isServer = typeof window === 'undefined';
  const { data, error } = useSuspenseQuery<Config>(GET_CONFIG, {
    skip: isServer,
  });

  useEffect(() => {
    if (error) {
      console.error('Error fetching config:', error);
    }
  }, [error]);

  return { data: data?.config, error };
};
