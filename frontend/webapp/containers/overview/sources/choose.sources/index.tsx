'use client';
import React from 'react';
import { useNotification } from '@/hooks';
import { useRouter } from 'next/navigation';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { OverviewHeader } from '@/components/overview';
import { NewSourcesList } from './new.source.flow';

export function SelectSourcesContainer() {
  const router = useRouter();
  const { Notification } = useNotification();

  function onNewSourceSuccess() {
    setTimeout(() => {
      router.push(`${ROUTES.SOURCES}?status=created`);
    }, 1000);
  }

  return (
    <div style={{ height: '100vh' }}>
      <OverviewHeader
        title={OVERVIEW.MENU.SOURCES}
        onBackClick={() => router.back()}
      />
      <NewSourcesList onSuccess={onNewSourceSuccess} />
      <Notification />
    </div>
  );
}
