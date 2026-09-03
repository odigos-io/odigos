import { ROUTES } from '../constants';
import { SVG } from '@odigos/ui-kit/types';
import { NavbarProps } from '@odigos/ui-kit/components';
import { AppRouterInstance } from 'next/dist/shared/lib/app-router-context.shared-runtime';
import { InsightsIcon, OverviewIcon, PipelineCollectorIcon, SamplingIcon, ServiceMapIcon, SettingsIcon, UrlTemplatizationIcon } from '@odigos/ui-kit/icons';

const getPayloadForIcon = (router: AppRouterInstance, currentPath: string, targetPath: string, label: string, icon: SVG): NavbarProps['icons'][number] => {
  return {
    id: targetPath,
    label,
    icon,
    selected: currentPath === targetPath,
    onClick: () => router.push(targetPath),
  };
};

export const getNavbarIcons = (router: AppRouterInstance, currentPath: string, insightsEnabled?: boolean) => {
  const navIcons: NavbarProps['icons'] = [];

  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.OVERVIEW, 'Overview', OverviewIcon));

  // The odigos-insights service is only deployed when insights are enabled in the
  // effective config, so the entry stays hidden until the feature is actually there.
  if (insightsEnabled) {
    navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.INSIGHTS, 'Insights', InsightsIcon));
  }

  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.SERVICE_MAP, 'Service Map', ServiceMapIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.PIPELINE_COLLECTORS, 'Collectors Pipeline', PipelineCollectorIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.SAMPLING, 'Sampling Rules', SamplingIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.URL_TEMPLATIZATION, 'URL Templatization', UrlTemplatizationIcon));
  navIcons.push(getPayloadForIcon(router, currentPath, ROUTES.SETTINGS, 'Settings', SettingsIcon));

  return navIcons;
};
