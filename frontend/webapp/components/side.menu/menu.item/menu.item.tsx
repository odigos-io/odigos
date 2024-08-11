import { KeyvalText } from '@/design.system';
import React, { useEffect, useState } from 'react';
import { styled } from 'styled-components';
interface MenuItemContainerProps {
  focused?: boolean;
  expand?: boolean;
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
  expand?: boolean;
}

const MenuItemContainer = styled.div<MenuItemContainerProps>`
  display: flex;
  cursor: pointer;
  width: ${({ expand }) => (expand ? '100%' : '48px')};
  border-radius: 10px;
  background: ${({ focused, theme }) =>
    focused ? theme.colors.blue_grey : 'transparent'};
  margin-bottom: 4px;
  &:hover {
    background: ${({ theme }) => theme.colors.blue_grey};
  }
`;

const TextWrapper = styled.div<MenuItemContainerProps>`
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  left: -5%;
  opacity: 0;
  animation: slideInFromLeft 0.5s forwards;
  @keyframes slideInFromLeft {
    to {
      left: ${({ expand }) => (expand ? '0' : '-10%')};
      opacity: ${({ expand }) => (expand ? '1' : '0')};
    }
  }
`;

const IconWrapper = styled.div<MenuItemContainerProps>`
  height: 48px;
  width: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
`;

export default function MenuItem({
  item,
  focused,
  onClick,
  expand = false,
}: MenuItemProps) {
  const { name, icons } = item;

  const [showText, setShowText] = useState(false);
  let timeout: NodeJS.Timeout;
  useEffect(() => {
    onExpandChange();
    return () => clearTimeout(timeout);
  }, [expand]);

  function onExpandChange() {
    if (expand) {
      timeout = setTimeout(() => {
        setShowText(true);
      }, 20);
    } else {
      setShowText(false);
    }
  }

  const iconToRender = focused ? icons.focus() : icons.notFocus();
  return (
    <MenuItemContainer data-cy={'menu-'+ name} onClick={onClick} focused={focused} expand={expand}>
      <IconWrapper>{iconToRender}</IconWrapper>
      {showText && (
        <TextWrapper expand={showText}>
          <KeyvalText size={14} weight={600}>
            {name}
          </KeyvalText>
        </TextWrapper>
      )}
    </MenuItemContainer>
  );
}
