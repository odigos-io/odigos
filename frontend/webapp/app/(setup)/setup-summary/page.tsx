'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { SetupSummary } from '@odigos/ui-kit/containers';

export default function Page() {
  const router = useRouter();

  return (
    <>
      <SetupHeader step={5} />
      <SetupSummary onEditSources={() => router.push(ROUTES.CHOOSE_SOURCES + '?skipToSummary=true')} onEditDestinations={() => router.push(ROUTES.CHOOSE_DESTINATION + '?skipToSummary=true')} />
    </>
  );
}
