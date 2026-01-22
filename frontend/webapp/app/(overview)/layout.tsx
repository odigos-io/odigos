'use client';

import React, { CSSProperties, useCallback, useMemo, type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import styled from 'styled-components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { DATA_FLOW_HEIGHT, MENU_BAR_HEIGHT, ROUTES } from '@/utils';
import { ErrorBoundary, FlexColumn } from '@odigos/ui-kit/components';
import { OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { PipelineCollectorIcon, ServiceMapIcon } from '@odigos/ui-kit/icons';
import { useConfig, useDataStreamsCRUD, useSSE, useTokenTracker } from '@/hooks';
import { DataFlowActionsMenu, NavIconIds, SideNav, ToastList } from '@odigos/ui-kit/containers';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors?.primary};
  align-items: center;
`;

const ContentWithActions = styled.div<{ $height: CSSProperties['height'] }>`
  width: 100%;
  height: ${({ $height }) => $height};
  position: relative;
`;

const ContentUnderActions = styled.div`
  gap: 12px;
  display: flex;
  justify-content: space-between;
  padding: 0 12px;
  width: calc(100% - 24px);
`;

const serviceMapId = 'service-map';
const serviceMapDisplayName = 'Service Map';

const pipelineCollectorsId = 'pipeline-collectors';
const pipelineCollectorsDisplayName = 'Pipeline Collectors';

const getEntityType = (pathname: string) => {
  return pathname.includes(ROUTES.SOURCES)
    ? EntityTypes.Source
    : pathname.includes(ROUTES.DESTINATIONS)
    ? EntityTypes.Destination
    : pathname.includes(ROUTES.ACTIONS)
    ? EntityTypes.Action
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? EntityTypes.InstrumentationRule
    : undefined;
};

const getSelectedId = (pathname: string) => {
  return pathname.includes(ROUTES.OVERVIEW)
    ? NavIconIds.Overview
    : pathname.includes(ROUTES.SOURCES)
    ? NavIconIds.Sources
    : pathname.includes(ROUTES.DESTINATIONS)
    ? NavIconIds.Destinations
    : pathname.includes(ROUTES.ACTIONS)
    ? NavIconIds.Actions
    : pathname.includes(ROUTES.INSTRUMENTATION_RULES)
    ? NavIconIds.InstrumentationRules
    : pathname.includes(ROUTES.SERVICE_MAP)
    ? serviceMapId
    : pathname.includes(ROUTES.PIPELINE_COLLECTORS)
    ? pipelineCollectorsId
    : undefined;
};

const routesMap = {
  [NavIconIds.Overview]: ROUTES.OVERVIEW,
  [NavIconIds.Sources]: ROUTES.SOURCES,
  [NavIconIds.Destinations]: ROUTES.DESTINATIONS,
  [NavIconIds.Actions]: ROUTES.ACTIONS,
  [NavIconIds.InstrumentationRules]: ROUTES.INSTRUMENTATION_RULES,
  [serviceMapId]: ROUTES.SERVICE_MAP,
  [pipelineCollectorsId]: ROUTES.PIPELINE_COLLECTORS,
};

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const { updateDataStream, deleteDataStream } = useDataStreamsCRUD();

  const router = useRouter();
  const pathname = usePathname();
  const { config } = useConfig();

  const entityType = useMemo(() => getEntityType(pathname), [pathname]);
  const selectedId = useMemo(() => getSelectedId(pathname), [pathname]);

  const onClickId = useCallback(
    (navId: keyof typeof routesMap) => {
      const route = routesMap[navId];
      if (route) router.push(route);
    },
    [router],
  );

  return (
    <ErrorBoundary>
      <OdigosProvider platformType={config?.platformType} tier={config?.tier} version={config?.odigosVersion || ''}>
        <PageContent>
          <OverviewHeader />

          <ContentWithActions $height={DATA_FLOW_HEIGHT}>
            {selectedId !== serviceMapId ? (
              <DataFlowActionsMenu addEntity={entityType} onClickNewDataStream={() => router.push(ROUTES.CHOOSE_STREAM)} updateDataStream={updateDataStream} deleteDataStream={deleteDataStream} />
            ) : (
              <div style={{ height: `${MENU_BAR_HEIGHT}px` }} />
            )}

            <ContentUnderActions>
              <SideNav
                defaultSelectedId={selectedId}
                onClickId={onClickId}
                extendedNavIcons={[
                  {
                    id: serviceMapId,
                    icon: ServiceMapIcon,
                    selected: selectedId === serviceMapId,
                    onClick: () => onClickId(serviceMapId),
                    tooltip: serviceMapDisplayName,
                  },
                  {
                    id: pipelineCollectorsId,
                    icon: PipelineCollectorIcon,
                    selected: selectedId === pipelineCollectorsId,
                    onClick: () => onClickId(pipelineCollectorsId),
                    tooltip: pipelineCollectorsDisplayName,
                  },
                ]}
              />
              {children}
            </ContentUnderActions>
          </ContentWithActions>

          <OverviewModalsAndDrawers />
          <ToastList />
        </PageContent>
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
