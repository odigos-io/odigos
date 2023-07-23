import { OVERVIEW, ROUTES } from "@/utils/constants";
import * as ICONS from "../../../assets/icons/side.menu";
import { MenuItem } from "./menu";

export const MENU_ITEMS: MenuItem[] = [
  {
    id: 1,
    name: OVERVIEW.MENU.OVERVIEW,
    icons: {
      focus: () => <ICONS.FocusOverview />,
      notFocus: () => <ICONS.UnFocusOverview />,
    },
    navigate: ROUTES.OVERVIEW,
  },
  {
    id: 2,
    name: OVERVIEW.MENU.SOURCES,
    icons: {
      focus: () => <ICONS.FocusSources />,
      notFocus: () => <ICONS.UnFocusSources />,
    },
    navigate: ROUTES.SOURCES,
  },
  {
    id: 3,
    name: OVERVIEW.MENU.DESTINATIONS,
    icons: {
      focus: () => <ICONS.FocusDestinations />,
      notFocus: () => <ICONS.UnFocusDestinations />,
    },
    navigate: ROUTES.DESTINATIONS,
  },
];
