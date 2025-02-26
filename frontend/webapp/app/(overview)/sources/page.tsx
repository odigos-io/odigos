'use client';

import React from 'react';
import { useMetrics, useSourceCRUD } from '@/hooks';
import { MultiSourceControl, SourceTable } from '@odigos/ui-containers';

export default function Page() {
  const { metrics } = useMetrics();
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <SourceTable sources={sources} metrics={metrics} maxHeight='calc(100vh - 220px)' maxWidth='calc(100vw - 70px)' />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} />
    </>
  );
}
