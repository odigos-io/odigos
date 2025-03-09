'use client';

import React from 'react';
import { useMetrics } from '@/hooks';
import { DestinationTable } from '@odigos/ui-containers';

export default function Page() {
  const { metrics } = useMetrics();

  return <DestinationTable metrics={metrics} maxHeight='calc(100vh - 220px)' maxWidth='calc(100vw - 70px)' />;
}
