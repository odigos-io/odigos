'use client';
import React from 'react';
import { OVERVIEW, ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { NewSourcesList } from './new.source.list';
import { OverviewHeader } from '@/components/overview';

export function SelectSourcesContainer() {
  const router = useRouter();

  function onNewSourceSuccess() {
    router.push(`${ROUTES.SOURCES}?poll=true`);
  }

  return (
    <div style={{ height: '100vh' }}>
      <OverviewHeader
        title={OVERVIEW.ADD_NEW_SOURCE}
        onBackClick={() => router.back()}
      />
      <NewSourcesList onSuccess={onNewSourceSuccess} />
    </div>
  );
}
