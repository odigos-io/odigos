'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { OverviewHeader } from '@/components';
import { ManagedSourcesContainer } from '@/containers';

export default function SourcesPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.SOURCES} />
      <ManagedSourcesContainer />
    </>
  );
}
