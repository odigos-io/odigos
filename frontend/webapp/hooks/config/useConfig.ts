<<<<<<< HEAD
=======
'use client';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
import { useEffect } from 'react';
import { Config } from '@/types';
import { GET_CONFIG } from '@/graphql';
import { useSuspenseQuery } from '@apollo/client';

export const useConfig = () => {
<<<<<<< HEAD
  const { data, error } = useSuspenseQuery<Config>(GET_CONFIG);
=======
  const isServer = typeof window === 'undefined';
  const { data, error } = useSuspenseQuery<Config>(GET_CONFIG, {
    skip: isServer,
  });
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

  useEffect(() => {
    if (error) {
      console.error('Error fetching config:', error);
    }
  }, [error]);

  return { data: data?.config, error };
};
