'use client';
import React, { useEffect } from 'react';
import {
  NOTIFICATION,
  OVERVIEW,
  PARAMS,
  QUERIES,
  ROUTES,
} from '@/utils/constants';
import { OverviewHeader } from '@/components/overview';
import { SourcesContainerWrapper } from './sources.styled';
import { ManageSources } from './manage.sources';
import { useQuery } from 'react-query';
import { getSources } from '@/services';
import { useRouter, useSearchParams } from 'next/navigation';
import { useNotification } from '@/hooks';

export function InstrumentedSourcesContainer() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { show, Notification } = useNotification();
  const { data: sources, refetch: refetchSources } = useQuery(
    [QUERIES.API_SOURCES],
    getSources
  );
  useEffect(onPageLoad, [searchParams]);

  function getMessage(status: string) {
    switch (status) {
      case PARAMS.DELETED:
        return OVERVIEW.SOURCE_DELETED_SUCCESS;
      case PARAMS.CREATED:
        return OVERVIEW.SOURCE_CREATED_SUCCESS;
      case PARAMS.UPDATED:
        return OVERVIEW.SOURCE_UPDATE_SUCCESS;
      default:
        return '';
    }
  }

  function onPageLoad() {
    console.log({ sources });
    const status = searchParams.get(PARAMS.STATUS);
    if (status) {
      refetchSources();
      show({
        type: NOTIFICATION.SUCCESS,
        message: getMessage(status),
      });
      router.replace(ROUTES.SOURCES);
    }
  }

  return (
    <SourcesContainerWrapper>
      <OverviewHeader title={OVERVIEW.MENU.SOURCES} />
      <ManageSources
        onAddClick={() => router.push(ROUTES.CREATE_SOURCE)}
        sources={sources}
      />
      <Notification />
    </SourcesContainerWrapper>
  );
}
