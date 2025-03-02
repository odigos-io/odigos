'use client';

import React from 'react';
import { DataFlow, MultiSourceControl } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useSourceCRUD } from '@/hooks';

export default function Page() {
  const { metrics } = useMetrics();
  const { actions, actionsLoading } = useActionCRUD();
  const { destinations, destinationsLoading } = useDestinationCRUD();
  const { sources, sourcesLoading, persistSources } = useSourceCRUD();
  const { instrumentationRules, instrumentationRulesLoading } = useInstrumentationRuleCRUD();

  return (
    <>
      <DataFlow
        heightToRemove='176px'
        sources={sources}
        sourcesLoading={sourcesLoading}
        destinations={destinations}
        destinationsLoading={destinationsLoading}
        actions={actions}
        actionsLoading={actionsLoading}
        instrumentationRules={instrumentationRules}
        instrumentationRulesLoading={instrumentationRulesLoading}
        metrics={metrics}
      />
      <MultiSourceControl totalSourceCount={sources.length} uninstrumentSources={(payload) => persistSources(payload, {})} />
    </>
  );
}
