"use client";
import React, { useState } from "react";
import { MenuContainer, LogoWrapper, MenuItemsWrapper } from "./menu.styled";
import { KeyvalText } from "@/design.system";
import MenuItem from "../menu.item/menu.item";
import { useRouter } from "next/navigation";

import FocusOverview from "../../../assets/icons/focus-overview.svg";
import UnFocusOverview from "../../../assets/icons/unfocus-overview.svg";
import FocusSources from "../../../assets/icons/sources-focus.svg";
import UnFocusSources from "../../../assets/icons/sources-unfocus.svg";
import FocusDestinations from "../../../assets/icons/destinations-focus.svg";
import UnFocusDestinations from "../../../assets/icons/destinations-unfocus.svg";
import { OVERVIEW } from "@/utils/constants";
const MENU_ITEMS = [
  {
    id: 1,
    name: "Overview",
    icons: {
      focus: () => <FocusOverview />,
      notFocus: () => <UnFocusOverview />,
    },
    navigate: "/overview",
  },
  {
    id: 2,
    name: "Sources",
    icons: {
      focus: () => <FocusSources />,
      notFocus: () => <UnFocusSources />,
    },
    navigate: "/overview/sources",
  },
  {
    id: 3,
    name: "Destinations",
    icons: {
      focus: () => <FocusDestinations />,
      notFocus: () => <UnFocusDestinations />,
    },
    navigate: "/overview/destinations",
  },
];

export function Menu() {
  const [currentMenuItem, setCurrentMenuItem] = useState(MENU_ITEMS[0]);
  const router = useRouter();

  function handleMenuItemClick(item) {
    setCurrentMenuItem(item);
    router.push(item?.navigate);
  }

  function renderMenuItemsList() {
    return MENU_ITEMS.map((item) => (
      <MenuItem
        key={`${item.id}_${item.name}`}
        onClick={() => handleMenuItemClick(item)}
        focused={currentMenuItem?.id === item.id}
        item={item}
      />
    ));
  }

  return (
    <MenuContainer>
      <LogoWrapper>
        <KeyvalText size={32} weight={700}>
          {OVERVIEW.ODIGOS}
        </KeyvalText>
      </LogoWrapper>
      <MenuItemsWrapper>{renderMenuItemsList()}</MenuItemsWrapper>
    </MenuContainer>
  );
}
