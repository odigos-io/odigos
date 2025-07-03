'use client';

import React from 'react';
import { HEADER_HEIGHT, MENU_BAR_HEIGHT } from '@/utils';
import { useMetrics, useSourceCRUD, useWorkloadUtils } from '@/hooks';
import { DataFlow, MultiSourceControl } from '@odigos/ui-kit/containers';

export default function Page() {
  const { metrics } = useMetrics();
  const { restartWorkloads } = useWorkloadUtils();
  const { sources, persistSources } = useSourceCRUD();

  return (
    <>
      <DataFlow heightToRemove={HEADER_HEIGHT + MENU_BAR_HEIGHT} metrics={metrics} />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} restartWorkloads={restartWorkloads} />
    </>
  );
}
