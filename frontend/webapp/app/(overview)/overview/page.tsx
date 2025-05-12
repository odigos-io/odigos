'use client';

import React from 'react';
import { OVERVIEW_HEIGHT_WITHOUT_DATA_FLOW } from '@/utils';
import { useDataStreamsCRUD, useMetrics, useSourceCRUD } from '@/hooks';
import { DataFlow, MultiSourceControl } from '@odigos/ui-kit/containers';

export default function Page() {
  const { metrics } = useMetrics();
  const { dataStreams } = useDataStreamsCRUD();
  const { sources, persistSources } = useSourceCRUD();

  // TODO: pass this to DataFlow as prop
  console.log('dataStreams', dataStreams);

  return (
    <>
      <DataFlow heightToRemove={OVERVIEW_HEIGHT_WITHOUT_DATA_FLOW} metrics={metrics} />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} />
    </>
  );
}
