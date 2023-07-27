"use client";
import React, { useEffect, useState } from "react";
import { MenuContainer, LogoWrapper, MenuItemsWrapper } from "./menu.styled";
import { KeyvalText } from "@/design.system";
import MenuItem from "../menu.item/menu.item";
import { useRouter } from "next/navigation";
import { OVERVIEW } from "@/utils/constants";
import { MENU_ITEMS } from "./items";

export interface MenuItem {
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

  useEffect(onLoad, []);

  function onLoad() {
    const currentItem = MENU_ITEMS.find((item) => {
      return item.navigate === window.location.pathname;
    });
    if (currentItem?.id !== currentMenuItem.id) {
      handleMenuItemClick(currentItem);
    }
  }

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
