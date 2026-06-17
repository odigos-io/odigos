'use client';

import React from 'react';
import { ServiceMap } from '@odigos/ui-kit/containers';

// Polling lives inside `<ServiceMap>` via `useOdigosApi()`.
export default function Page() {
  return <ServiceMap />;
}
