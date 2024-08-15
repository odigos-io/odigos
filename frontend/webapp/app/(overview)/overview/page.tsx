'use client';
import React, { useEffect } from 'react';
import { OVERVIEW } from '@/utils';
import { OverviewHeader } from '@/components';
import { DataFlowContainer } from '@/containers';
import { useQuery } from '@apollo/client';
import { GET_COMPUTE_PLATFORM } from '@/graphql';

export default function OverviewPage() {
  const { data, loading, error } = useQuery(GET_COMPUTE_PLATFORM);

  useEffect(() => {
    if (error) {
      console.error(error);
    }
    if (data) {
      console.log(data);
    }
  }, [error, data]);

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.OVERVIEW} />
      <DataFlowContainer />
    </>
  );
}
