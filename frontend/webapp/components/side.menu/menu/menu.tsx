"use client";
import React, { useState } from "react";
import { MenuContainer, LogoWrapper, MenuItemsWrapper } from "./menu.styled";
import { KeyvalText } from "@/design.system";
import MenuItem from "../menu.item/menu.item";
import { useRouter } from "next/navigation";
import { OVERVIEW, ROUTES } from "@/utils/constants";
import * as ICONS from "../../../assets/icons/side.menu";

const MENU_ITEMS = [
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

interface MenuItem {
  id: number;
  name: string;
  icons: {
    focus: () => JSX.Element;
    notFocus: () => JSX.Element;
  };
  navigate: string;
}

export function Menu() {
  const [currentMenuItem, setCurrentMenuItem] = useState<MenuItem>(
    MENU_ITEMS[0]
  );
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
