'use client';

import React from 'react';
import { DestinationTable } from '@odigos/ui-containers';
import { useDestinationCRUD, useMetrics } from '@/hooks';

export default function Page() {
  const { metrics } = useMetrics();
  const { destinations } = useDestinationCRUD();

  return <DestinationTable destinations={destinations} metrics={metrics} maxHeight='calc(100vh - 220px)' maxWidth='calc(100vw - 70px)' />;
}
