import { OVERVIEW, ROUTES } from '@/utils/constants';
import { MenuItem } from './menu';
import {
  FocusActionIcon,
  FocusDestinationsIcon,
  FocusOverviewIcon,
  FocusSourcesIcon,
  UnFocusActionIcon,
  UnFocusDestinationsIcon,
  UnFocusOverviewIcon,
  UnFocusSourcesIcon,
} from '@keyval-dev/design-system';
import { Funnel, FunnelFocus } from '@/assets';

export const MENU_ITEMS: MenuItem[] = [
  {
    id: 1,
    name: OVERVIEW.MENU.OVERVIEW,
    icons: {
      focus: () => <FocusOverviewIcon />,
      notFocus: () => <UnFocusOverviewIcon />,
    },
    navigate: ROUTES.OVERVIEW,
  },
  {
    id: 2,
    name: OVERVIEW.MENU.SOURCES,
    icons: {
      focus: () => <FocusSourcesIcon />,
      notFocus: () => <UnFocusSourcesIcon />,
    },
    navigate: ROUTES.SOURCES,
  },
  {
    id: 3,
    name: OVERVIEW.MENU.INSTRUMENTATION_RULES,
    icons: {
      focus: () => <FunnelFocus style={{ width: 24 }} />,
      notFocus: () => <Funnel style={{ width: 24 }} />,
    },
    navigate: ROUTES.INSTRUMENTATION_RULES,
  },
  {
    id: 4,
    name: OVERVIEW.MENU.ACTIONS,
    icons: {
      focus: () => <FocusActionIcon />,
      notFocus: () => <UnFocusActionIcon />,
    },
    navigate: ROUTES.ACTIONS,
  },
  {
    id: 5,
    name: OVERVIEW.MENU.DESTINATIONS,
    icons: {
      focus: () => <FocusDestinationsIcon />,
      notFocus: () => <UnFocusDestinationsIcon />,
    },
    navigate: ROUTES.DESTINATIONS,
  },
];
