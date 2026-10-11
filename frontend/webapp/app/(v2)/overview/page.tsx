'use client';

import React, { useCallback } from 'react';
import { ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { Overview } from '@odigos/ui-kit/containers';

// All data fetching (metrics polling, effective config for the Profiling
// tab, drawer-fed entity refreshes) lives inside the kit's `<Overview>`
// container via `useOdigosApi()`. The page is a one-liner.
export default function Page() {
  const router = useRouter();

  // Fired by the Source Drawer's "open in Sampling page" hover button.
  // The kit handles both the source-drawer close and staging which rule to
  // open (via `useSamplingDrawerStore`); the page's only job is to navigate.
  const onRedirectToSampling = useCallback(() => router.push(ROUTES.SAMPLING), [router]);

  return <Overview onRedirectToSampling={onRedirectToSampling} />;
}
