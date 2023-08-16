import { KeyvalText } from '@/design.system';
import React from 'react';
import { styled } from 'styled-components';
interface MenuItemContainerProps {
  focused: boolean;
}

interface MenuItem {
  name: string;
  icons: {
    focus: () => JSX.Element;
    notFocus: () => JSX.Element;
  };
}

interface MenuItemProps {
  item: MenuItem;
  focused: boolean;
  onClick?: () => void;
}

const MenuItemContainer = styled.div<MenuItemContainerProps>`
  display: flex;
  padding: 0px 16px;
  height: 48px;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  border-radius: 10px;
  background: ${({ focused, theme }) =>
    focused ? theme.colors.blue_grey : 'transparent'};
`;

export default function MenuItem({ item, focused, onClick }: MenuItemProps) {
  const { name, icons } = item;

  return (
    <MenuItemContainer onClick={onClick} focused={focused}>
      {focused ? icons.focus() : icons.notFocus()}
      <KeyvalText size={14} weight={600}>
        {name}
      </KeyvalText>
    </MenuItemContainer>
  );
}
