'use client';
import React, { useEffect, useState } from 'react';
import {
  MenuContainer,
  LogoWrapper,
  MenuItemsWrapper,
  ContactUsWrapper,
} from './menu.styled';
import { KeyvalImage, KeyvalText } from '@/design.system';
import MenuItem from '../menu.item/menu.item';
import { useRouter } from 'next/navigation';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { MENU_ITEMS } from './items';
import ContactUsButton from '../contact.us/contact.us';

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
  const [isHovered, setIsHovered] = useState(false);
  const router = useRouter();

  useEffect(onLoad, []);

  function onLoad() {
    const currentItem = MENU_ITEMS.find(
      ({ navigate }) =>
        navigate !== ROUTES.OVERVIEW &&
        window.location.pathname.includes(navigate)
    );

    currentItem && setCurrentMenuItem(currentItem);
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
        expand={isHovered}
      />
    ));
  }

  return (
    <MenuContainer
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <LogoWrapper>
        {isHovered ? (
          <KeyvalText size={32} weight={700}>
            {OVERVIEW.ODIGOS}
          </KeyvalText>
        ) : (
          <KeyvalImage
            src={'https://d2q89wckrml3k4.cloudfront.net/logo.png'}
            width={40}
            height={40}
          />
        )}
      </LogoWrapper>
      <MenuItemsWrapper>{renderMenuItemsList()}</MenuItemsWrapper>
      <ContactUsWrapper>
        <ContactUsButton expand={isHovered} />
      </ContactUsWrapper>
    </MenuContainer>
  );
}
