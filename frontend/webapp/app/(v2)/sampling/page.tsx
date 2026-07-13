'use client';

import React from 'react';
import { SamplingRules } from '@odigos/ui-kit/containers';

// Sampling rules + k8s health probes config + workload list now flow via
// `useOdigosApi()` inside the kit container.
export default function Page() {
  return <SamplingRules />;
}
