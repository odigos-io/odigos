'use client';

import React from 'react';
import { useMetrics, useSourceCRUD } from '@/hooks';
import { DataFlow, MultiSourceControl } from '@odigos/ui-containers';

export default function Page() {
  const { metrics } = useMetrics();
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <DataFlow heightToRemove='176px' metrics={metrics} />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} />
    </>
  );
}
