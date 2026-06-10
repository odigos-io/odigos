'use client';

import React from 'react';
import { Overview } from '@odigos/ui-kit/containers/v2';

// All data fetching (metrics polling, effective config for the Profiling
// tab, drawer-fed entity refreshes) lives inside the kit's `<Overview>`
// container via `useOdigosApi()`. The page is a one-liner.
export default function Page() {
  return <Overview />;
}
