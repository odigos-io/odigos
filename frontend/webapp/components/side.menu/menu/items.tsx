import { OVERVIEW, ROUTES } from '@/utils/constants';
import * as ICONS from '../../../assets/icons/side.menu';
import { MenuItem } from './menu';

export const MENU_ITEMS: MenuItem[] = [
  {
    id: 1,
    name: OVERVIEW.MENU.OVERVIEW,
    icons: {
      focus: () => <ICONS.FocusOverview width={24} height={24} />,
      notFocus: () => <ICONS.UnFocusOverview width={24} height={24} />,
    },
    navigate: ROUTES.OVERVIEW,
  },
  {
    id: 2,
    name: OVERVIEW.MENU.SOURCES,
    icons: {
      focus: () => <ICONS.FocusSources width={24} height={24} />,
      notFocus: () => <ICONS.UnFocusSources width={24} height={24} />,
    },
    navigate: ROUTES.SOURCES,
  },
  {
    id: 3,
    name: OVERVIEW.MENU.DESTINATIONS,
    icons: {
      focus: () => <ICONS.FocusDestinations width={24} height={24} />,
      notFocus: () => <ICONS.UnFocusDestinations width={24} height={24} />,
    },
    navigate: ROUTES.DESTINATIONS,
  },
];
