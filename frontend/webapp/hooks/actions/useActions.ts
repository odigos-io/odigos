import { getActions } from '@/services';
import { QUERIES } from '@/utils';
import React, { useEffect } from 'react';
import { useQuery } from 'react-query';

export function useActions() {
  const { isLoading, data } = useQuery([QUERIES.API_ACTIONS], getActions);

  useEffect(() => {
    console.log({ data });
  }, [data]);

  return {
    isLoading,
    actions: data || [],
  };
}
