'use client';
import React from 'react';
import { useQuery } from 'react-query';
import { getSources } from '@/services';
import { useNotification } from '@/hooks';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components/overview';
import { OVERVIEW, QUERIES, ROUTES } from '@/utils/constants';
import { NewSourcesList } from '@/containers/overview/sources/new.source.flow';

export function SourcesListContainer() {
  const { Notification } = useNotification();
  const router = useRouter();
  const { data: sources } = useQuery([QUERIES.API_SOURCES], getSources);

  function onNewSourceSuccess() {
    setTimeout(() => {
      router.push(`${ROUTES.SOURCES}?status=created`);
    }, 1000);
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.SOURCES}
        onBackClick={() => router.back()}
      />
      <NewSourcesList onSuccess={onNewSourceSuccess} sources={sources} />
      <Notification />
    </>
  );
}
