import React from 'react';
import { Button } from '..';
import Image from 'next/image';
import styled from 'styled-components';
<<<<<<< HEAD
=======
import { hexPercentValues } from '@/styles/theme';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
import { Badge, Text } from '@/reuseable-components';

interface Props {
  label: string;
<<<<<<< HEAD
  badgeLabel?: string | number;
  icon?: string;
  isSelected?: boolean;
  onClick: () => void;
=======
  onClick: () => void;
  icon?: string;
  badgeLabel?: string | number;
  badgeFilled?: boolean;
  isSelected?: boolean;
  withBorder?: boolean;
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  color?: React.CSSProperties['backgroundColor'];
  hoverColor?: React.CSSProperties['backgroundColor'];
  style?: React.CSSProperties;
}

<<<<<<< HEAD
const StyledButton = styled(Button)<{ color: Props['color']; hoverColor: Props['hoverColor'] }>`
  gap: 8px;
  text-transform: none;
  text-decoration: none;
  border: none;
=======
const StyledButton = styled(Button)<{ withBorder: Props['withBorder']; color: Props['color']; hoverColor: Props['hoverColor'] }>`
  gap: 8px;
  text-transform: none;
  text-decoration: none;
  border: ${({ theme, withBorder }) => `1px solid ${withBorder ? theme.colors.border : 'transparent'}`};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  background-color: ${({ theme, color }) => color || theme.colors.white_opacity['004']};
  &.not-selected {
    &:hover {
      background-color: ${({ theme, hoverColor }) => hoverColor || theme.colors.white_opacity['008']};
    }
  }
  &.selected {
<<<<<<< HEAD
    background-color: ${({ theme }) => theme.colors.majestic_blue + '7A'};
  }
`;

export const SelectionButton = ({ label, badgeLabel, icon, isSelected, onClick, color, hoverColor, style }: Props) => {
  return (
    <StyledButton onClick={onClick} className={isSelected ? 'selected' : 'not-selected'} color={color} hoverColor={hoverColor} style={style}>
=======
    background-color: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['048']};
  }
`;

export const SelectionButton = ({ label, onClick, icon, badgeLabel, badgeFilled, isSelected, withBorder, color, hoverColor, style }: Props) => {
  return (
    <StyledButton onClick={onClick} className={isSelected ? 'selected' : 'not-selected'} withBorder={withBorder} color={color} hoverColor={hoverColor} style={style}>
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
      {icon && <Image src={icon} alt='' width={16} height={16} />}
      <Text size={14} style={{ whiteSpace: 'nowrap' }}>
        {label}
      </Text>
<<<<<<< HEAD
      {badgeLabel !== undefined && <Badge label={badgeLabel} filled={isSelected} />}
=======
      {badgeLabel !== undefined && <Badge label={badgeLabel} filled={badgeFilled || isSelected} />}
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
    </StyledButton>
  );
};
