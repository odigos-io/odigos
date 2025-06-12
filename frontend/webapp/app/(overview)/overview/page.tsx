'use client';

import React from 'react';
import { OVERVIEW_HEIGHT_WITHOUT_DATA_FLOW } from '@/utils';
import { useMetrics, useSourceCRUD, useWorkloadUtils } from '@/hooks';
import { DataFlow, MultiSourceControl } from '@odigos/ui-kit/containers';

export default function Page() {
  const { metrics } = useMetrics();
  const { restartWorkloads } = useWorkloadUtils();
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <DataFlow heightToRemove={OVERVIEW_HEIGHT_WITHOUT_DATA_FLOW} metrics={metrics} />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} restartWorkloads={restartWorkloads} />
    </>
  );
}
