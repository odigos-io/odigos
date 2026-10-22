'use client';

import React, { Suspense } from 'react';
import { TelemetryPage } from '@/components/telemetry';

export default function Page() {
  return (
    <Suspense fallback={null}>
      <TelemetryPage />
    </Suspense>
  );
}
