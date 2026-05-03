import { ROUTES } from '../constants';
import { SVG } from '@odigos/ui-kit/types';
import { NavbarProps } from '@odigos/ui-kit/components/v2';
import { AppRouterInstance } from 'next/dist/shared/lib/app-router-context.shared-runtime';
import { OverviewIcon, PipelineCollectorIcon, ServiceMapIcon, SettingsIcon } from '@odigos/ui-kit/icons';

const getPayloadForIcon = (router: AppRouterInstance, currentPath: string, targetPath: string, label: string, icon: SVG): NavbarProps['icons'][number] => {
  return {
    id: targetPath,
    label,
    icon,
    selected: currentPath === targetPath,
    onClick: () => router.push(targetPath),
  };
};

export const getNavbarIcons = (router: AppRouterInstance, currentPath: string) => {
  const navIcons: NavbarProps['icons'] = [
    getPayloadForIcon(router, currentPath, ROUTES.OVERVIEW, 'Overview', OverviewIcon),
    getPayloadForIcon(router, currentPath, ROUTES.SERVICE_MAP, 'Service Map', ServiceMapIcon),
    getPayloadForIcon(router, currentPath, ROUTES.PIPELINE_COLLECTORS, 'Collectors Pipeline', PipelineCollectorIcon),
    // getPayloadForIcon(router, currentPath, ROUTES.SAMPLING, 'Sampling Rules', SamplingIcon),
    getPayloadForIcon(router, currentPath, ROUTES.SETTINGS, 'Settings', SettingsIcon),
  ];

  return navIcons;
};
