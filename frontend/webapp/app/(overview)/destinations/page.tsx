'use client';

import React from 'react';
import { useMetrics } from '@/hooks';
import { TABLE_MAX_HEIGHT, TABLE_MAX_WIDTH } from '@/utils';
import { DestinationTable } from '@odigos/ui-kit/containers';

export default function Page() {
  const { metrics } = useMetrics();

  return <DestinationTable metrics={metrics} maxHeight={TABLE_MAX_HEIGHT} maxWidth={TABLE_MAX_WIDTH} />;
}
