import { ROUTES } from '../constants';
import { SVG } from '@odigos/ui-kit/types';
import { NavbarProps } from '@odigos/ui-kit/components/v2';
import { AppRouterInstance } from 'next/dist/shared/lib/app-router-context.shared-runtime';
import { ActionIcon, DestinationIcon, InstrumentationRuleIcon, OverviewIcon, PipelineCollectorIcon, ServiceMapIcon, SourceIcon } from '@odigos/ui-kit/icons';

const getPayloadForIcon = (router: AppRouterInstance, currentPath: string, targetPath: string, icon: SVG): NavbarProps['icons'][number] => {
  return {
    id: targetPath,
    icon,
    selected: currentPath === targetPath,
    onClick: () => router.push(targetPath),
  };
};

export const getNavbarIcons = (router: AppRouterInstance, currentPath: string) => {
  const navIcons: NavbarProps['icons'] = [];

  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.OVERVIEW, OverviewIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.SOURCES, SourceIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.DESTINATIONS, DestinationIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.ACTIONS, ActionIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.INSTRUMENTATION_RULES, InstrumentationRuleIcon));

  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.SERVICE_MAP, ServiceMapIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.PIPELINE_COLLECTORS, PipelineCollectorIcon));

  return navIcons;
};
