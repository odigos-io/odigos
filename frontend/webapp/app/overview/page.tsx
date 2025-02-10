'use client';

import React from 'react';
import { MainContent } from '@/components';
import { usePaginatedStore } from '@/store';
import type { Source } from '@odigos/ui-utils';
import OverviewHeader from '@/components/lib-imports/overview-header';
import OverviewModalsAndDrawers from '@/components/lib-imports/overview-modals-and-drawers';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, type SourceSelectionFormData } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

export default function Page() {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const { sourcesFetching } = usePaginatedStore();

  const { metrics } = useMetrics();
  const { allNamespaces } = useNamespace();
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { sources, filteredSources, loading: srcLoad, persistSources } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  return (
    <>
      <OverviewHeader />
      <MainContent>
        <DataFlowActionsMenu namespaces={allNamespaces} sources={filteredSources} destinations={filteredDestinations} actions={filteredActions} instrumentationRules={filteredInstrumentationRules} />
        <DataFlow
          heightToRemove='176px'
          sources={filteredSources}
          sourcesLoading={srcLoad || sourcesFetching}
          sourcesTotalCount={sources.length}
          destinations={filteredDestinations}
          destinationsLoading={destLoad}
          destinationsTotalCount={destinations.length}
          actions={filteredActions}
          actionsLoading={actLoad}
          actionsTotalCount={actions.length}
          instrumentationRules={filteredInstrumentationRules}
          instrumentationRulesLoading={ruleLoad}
          instrumentationRulesTotalCount={instrumentationRules.length}
          metrics={metrics}
        />
        <MultiSourceControl
          totalSourceCount={sources.length}
          uninstrumentSources={(payload) => {
            const inp: SourceSelectionFormData = {};

            Object.entries(payload).forEach(([namespace, sources]: [string, Source[]]) => {
              inp[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
            });

            persistSources(inp, {});
          }}
        />
      </MainContent>
      <OverviewModalsAndDrawers />
    </>
  );
}
