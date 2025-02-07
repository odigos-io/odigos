'use client';
import React from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { usePaginatedStore } from '@/store';
import { type SourceInstrumentInput } from '@/types';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, Source } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

import { MainHeader } from '@/components';
const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });

const MainContent = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

export default function MainPage() {
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
      <MainHeader />
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
            const inp: SourceInstrumentInput = {};

            Object.entries(payload).forEach(([namespace, sources]: [string, Source[]]) => {
              inp[namespace] = sources.map(({ name, kind }) => ({ name, kind, selected: false }));
            });

            persistSources(inp, {});
          }}
        />
      </MainContent>

      <AllModals />
      <AllDrawers />
    </>
  );
}
