'use client';
import React from 'react';
import { OverviewHeader } from '@/components/overview';
import { OVERVIEW } from '@/utils/constants';
import { OverviewContainer } from '@/containers/overview';

export default function OverviewPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.OVERVIEW} />
      <OverviewContainer />
    </>
  );
}
