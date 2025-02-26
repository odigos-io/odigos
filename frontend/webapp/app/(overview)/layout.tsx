'use client';

import React, { type PropsWithChildren } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import styled from 'styled-components';
import { FlexColumn, FlexRow } from '@odigos/ui-components';
import { ErrorBoundary, OverviewHeader, OverviewModalsAndDrawers } from '@/components';
import { DataFlowActionsMenu, NAV_ICON_IDS, SideNav, ToastList } from '@odigos/ui-containers';
import { useActionCRUD, useDestinationCRUD, useInstrumentationRuleCRUD, useNamespace, useSourceCRUD, useSSE, useTokenTracker } from '@/hooks';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

const ContentWithActions = styled.div`
  width: 100%;
  height: calc(100vh - 176px);
  position: relative;
`;

const ContentUnderActions = styled(FlexRow)`
  align-items: flex-start;
  padding-left: 12px;
  width: calc(100% - 12px);
`;

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  const pathname = usePathname();
  const router = useRouter();

  const { sources } = useSourceCRUD();
  const { actions } = useActionCRUD();
  const { namespaces } = useNamespace();
  const { destinations } = useDestinationCRUD();
  const { instrumentationRules } = useInstrumentationRuleCRUD();

  return (
    <ErrorBoundary>
      <PageContent>
        <OverviewHeader />

        <ContentWithActions>
          <DataFlowActionsMenu namespaces={namespaces} sources={sources} destinations={destinations} actions={actions} instrumentationRules={instrumentationRules} />
          <ContentUnderActions>
            <SideNav
              defaultSelectedId={
                pathname === ROUTES.OVERVIEW
                  ? NAV_ICON_IDS.OVERVIEW
                  : pathname === ROUTES.OVERVIEW_SOURCES
                  ? NAV_ICON_IDS.SOURCES
                  : pathname === ROUTES.OVERVIEW_DESTINATIONS
                  ? NAV_ICON_IDS.DESTINATIONS
                  : pathname === ROUTES.OVERVIEW_ACTIONS
                  ? NAV_ICON_IDS.ACTIONS
                  : pathname === ROUTES.OVERVIEW_INSTRUMENTATION_RULES
                  ? NAV_ICON_IDS.INSTRUMENTATION_RULES
                  : undefined
              }
              onClickId={(id) => {
                switch (id) {
                  case NAV_ICON_IDS.OVERVIEW:
                    router.push(ROUTES.OVERVIEW);
                    break;
                  case NAV_ICON_IDS.SOURCES:
                    router.push(ROUTES.OVERVIEW_SOURCES);
                    break;
                  case NAV_ICON_IDS.DESTINATIONS:
                    router.push(ROUTES.OVERVIEW_DESTINATIONS);
                    break;
                  case NAV_ICON_IDS.ACTIONS:
                    router.push(ROUTES.OVERVIEW_ACTIONS);
                    break;
                  case NAV_ICON_IDS.INSTRUMENTATION_RULES:
                    router.push(ROUTES.OVERVIEW_INSTRUMENTATION_RULES);
                    break;
                  default:
                    console.warn('unhandled nav icon id', id);
                    break;
                }
              }}
            />
            {children}
          </ContentUnderActions>
        </ContentWithActions>

        <OverviewModalsAndDrawers />
        <ToastList />
      </PageContent>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
