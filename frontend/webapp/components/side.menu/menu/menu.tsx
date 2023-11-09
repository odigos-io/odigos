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
import { METADATA, OVERVIEW, ROUTES } from '@/utils/constants';
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
  const [isExpanded, setIsExpanded] = useState(false);
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
        expand={isExpanded}
      />
    ));
  }

  function renderMenuLogo() {
    return (
      <LogoWrapper 
        onClick={() => setIsExpanded(!isExpanded)}
      >
        {isExpanded ? (
          <KeyvalText size={32} weight={700}>
            {OVERVIEW.ODIGOS}
          </KeyvalText>
        ) : (
          <KeyvalImage src={METADATA.icons} width={40} height={40} />
        )}
      </LogoWrapper>
    );
  }

  function renderContactUsButton() {
    return (
      <ContactUsWrapper>
        <ContactUsButton expand={isExpanded} />
      </ContactUsWrapper>
    );
  }

  return (
    <MenuContainer $isExpanded={isExpanded} >
      {renderMenuLogo()}
      <MenuItemsWrapper>{renderMenuItemsList()}</MenuItemsWrapper>
      {renderContactUsButton()}
    </MenuContainer>
  );
}
