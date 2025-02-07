'use client';
import React from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { type SourceInstrumentInput } from '@/types';
import { DataFlow, DataFlowActionsMenu, MultiSourceControl, Source, ToastList } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNamespace, usePaginatedSources, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });

const Container = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

export default function MainPage() {
  useSSE();
  useTokenTracker();

  // "usePaginatedSources" is here to fetch sources just once
  // (hooks run on every mount, we don't want that for pagination)
  const { loading: pageLoading } = usePaginatedSources();

  const { metrics } = useMetrics();
  const { allNamespaces } = useNamespace();
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { sources, filteredSources, loading: srcLoad, persistSources } = useSourceCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  return (
    <>
      <Container>
        <ToastList />

        <DataFlowActionsMenu namespaces={allNamespaces} sources={filteredSources} destinations={filteredDestinations} actions={filteredActions} instrumentationRules={filteredInstrumentationRules} />
        <DataFlow
          heightToRemove='176px'
          sources={filteredSources}
          sourcesLoading={srcLoad || pageLoading}
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

        <AllModals />
        <AllDrawers />
      </Container>
    </>
  );
}
