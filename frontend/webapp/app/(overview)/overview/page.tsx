'use client';
import React from 'react';
import { OVERVIEW } from '@/utils/constants';
import { OverviewHeader } from '@/components';
import { DataFlowContainer } from '@/containers';

export default function OverviewPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.OVERVIEW} />
      <DataFlowContainer />
    </>
  );
}
