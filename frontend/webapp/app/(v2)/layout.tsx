'use client';

import React, { useCallback, type PropsWithChildren, useEffect } from 'react';
import { usePathname, useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { useTheme } from 'styled-components';
import { OverviewHeader } from '@/components';
import { useDarkMode } from '@odigos/ui-kit/store';
import { Navbar } from '@odigos/ui-kit/components/v2';
import { OdigosProvider } from '@odigos/ui-kit/contexts';
import { useConfig, useSSE, useTokenTracker } from '@/hooks';
import { NavIconIds, ToastList } from '@odigos/ui-kit/containers';
import { ErrorBoundary, FlexColumn, FlexRow } from '@odigos/ui-kit/components';
import { OverviewIcon, PipelineCollectorIcon, ServiceMapIcon } from '@odigos/ui-kit/icons';

const serviceMapId = 'service-map';
const pipelineCollectorsId = 'pipeline-collectors';

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
  const { config } = useConfig();

  // TODO: remove this after migration to v2
  const theme = useTheme();
  const { darkMode, setDarkMode } = useDarkMode();
  useEffect(() => {
    if (!darkMode) setDarkMode(true);
    document.body.style.backgroundColor = theme.v2.colors.black['500'];
  }, [theme]);

  const router = useRouter();
  const pathname = usePathname();

  const navbarIcons = useCallback(
    (activeId: string) => {
      const onClickId = (navId: keyof typeof routesMap) => {
        const route = routesMap[navId];
        if (route) router.push(route);
      };

      return [
        {
          id: NavIconIds.Overview,
          icon: OverviewIcon,
          selected: activeId === NavIconIds.Overview,
          onClick: () => onClickId(NavIconIds.Overview),
        },
        {
          id: serviceMapId,
          icon: ServiceMapIcon,
          selected: activeId === serviceMapId,
          onClick: () => onClickId(serviceMapId),
        },
        {
          id: pipelineCollectorsId,
          icon: PipelineCollectorIcon,
          selected: activeId === pipelineCollectorsId,
          onClick: () => onClickId(pipelineCollectorsId),
        },
      ];
    },
    [router],
  );

  return (
    <ErrorBoundary>
      <OdigosProvider tier={config?.tier} version={config?.odigosVersion} platformType={config?.platformType}>
        <FlexColumn $gap={0}>
          <OverviewHeader v2 />
          <FlexRow $gap={0}>
            <Navbar height='calc(100vh - 60px)' icons={navbarIcons(getSelectedId(pathname) || '')} />
            {children}
          </FlexRow>
        </FlexColumn>

        <ToastList />
      </OdigosProvider>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
