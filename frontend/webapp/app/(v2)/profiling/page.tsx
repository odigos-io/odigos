'use client';

import React, { Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { Profiling } from '@odigos/ui-kit/containers';
import { ROUTES } from '@/utils/constants';

function ProfilingPage() {
  const router = useRouter();
  const searchParams = useSearchParams();

  return (
    <Profiling
      className='min-w-0 flex-1'
      unavailableSourceKey={searchParams.get('unavailableSource')}
      onUnavailableSourceDismiss={() => router.replace(ROUTES.PROFILING)}
      selectedSourceKey={searchParams.get('source')}
      onSelectedSourceKeyChange={(key) => router.push(key ? `${ROUTES.PROFILING}?${new URLSearchParams({ source: key })}` : ROUTES.PROFILING)}
    />
  );
}

export default function Page() {
  return (
    <Suspense>
      <ProfilingPage />
    </Suspense>
  );
}
