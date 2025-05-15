'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES, SKIP_TO_SUMMERY_QUERY_PARAM } from '@/utils';
import { SetupHeader } from '@/components';
import { SetupSummary } from '@odigos/ui-kit/containers';

export default function Page() {
  const router = useRouter();
  const queryString = `?${SKIP_TO_SUMMERY_QUERY_PARAM}=true`;

  return (
    <>
      <SetupHeader step={5} />
      <SetupSummary onEditSources={() => router.push(ROUTES.CHOOSE_SOURCES + queryString)} onEditDestinations={() => router.push(ROUTES.CHOOSE_DESTINATION + queryString)} />
    </>
  );
}
