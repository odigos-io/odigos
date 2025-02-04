'use client';
import React from 'react';
import dynamic from 'next/dynamic';
import styled from 'styled-components';
import { DataFlow } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useMetrics, useNodeDataFlowHandlers, usePaginatedSources, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

const ToastList = dynamic(() => import('@/components/notification/toast-list'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });
const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });

const OverviewActionsMenu = dynamic(() => import('@/containers/main/overview/overview-actions-menu'), { ssr: false });
const MultiSourceControl = dynamic(() => import('@/containers/main/overview/multi-source-control'), { ssr: false });

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

  const { handleNodeClick } = useNodeDataFlowHandlers();

  const { metrics } = useMetrics();
  const { sources, filteredSources, loading: srcLoad } = useSourceCRUD();
  const { actions, filteredActions, loading: actLoad } = useActionCRUD();
  const { destinations, filteredDestinations, loading: destLoad } = useDestinationCRUD();
  const { instrumentationRules, filteredInstrumentationRules, loading: ruleLoad } = useInstrumentationRuleCRUD();

  return (
    <>
      <ToastList />
      <MultiSourceControl />
      <AllDrawers />
      <AllModals />

      <Container>
        <OverviewActionsMenu />
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
          onNodeClick={handleNodeClick}
        />
      </Container>
    </>
  );
}
