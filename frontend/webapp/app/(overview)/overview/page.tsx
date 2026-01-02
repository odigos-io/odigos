'use client';

import React from 'react';
import { HEADER_HEIGHT, MENU_BAR_HEIGHT } from '@/utils';
import { DataFlow, MultiSourceControl } from '@odigos/ui-kit/containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useSourceCRUD, useWorkloadUtils } from '@/hooks';

export default function Page() {
  const { metrics } = useMetrics();
  const { fetchActions } = useActionCRUD();
  const { restartWorkloads } = useWorkloadUtils();
  const { fetchDestinations } = useDestinationCRUD();
  const { sources, persistSources, fetchSources } = useSourceCRUD();
  const { fetchInstrumentationRules } = useInstrumentationRuleCRUD();

  return (
    <>
      <DataFlow
        height={`calc(100vh - ${HEADER_HEIGHT + MENU_BAR_HEIGHT + 100}px)`}
        metrics={metrics}
        refetchSources={fetchSources}
        refetchDestinations={fetchDestinations}
        refetchActions={fetchActions}
        refetchInstrumentationRules={fetchInstrumentationRules}
      />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} restartWorkloads={restartWorkloads} />
    </>
  );
}
