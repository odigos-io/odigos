import React from 'react';
import { Button } from '..';
import Image from 'next/image';
import styled from 'styled-components';
import { hexPercentValues } from '@/styles/theme';
import { Badge, Text } from '@/reuseable-components';

interface Props {
  label: string;
  onClick: () => void;
  icon?: string;
  badgeLabel?: string | number;
  badgeFilled?: boolean;
  isSelected?: boolean;
  withBorder?: boolean;
  color?: React.CSSProperties['backgroundColor'];
  hoverColor?: React.CSSProperties['backgroundColor'];
  style?: React.CSSProperties;
}

const StyledButton = styled(Button)<{ $withBorder: Props['withBorder']; $color: Props['color']; $hoverColor: Props['hoverColor'] }>`
  gap: 8px;
  text-transform: none;
  text-decoration: none;
  border: ${({ theme, $withBorder }) => `1px solid ${$withBorder ? theme.colors.border : 'transparent'}`};
  &.not-selected {
    background-color: ${({ theme, $color }) => $color || theme.colors.white_opacity['004']};
    &:hover {
      background-color: ${({ theme, $hoverColor }) => $hoverColor || theme.colors.white_opacity['008']};
    }
  }
  &.selected {
    background-color: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['048']};
  }
`;

export const SelectionButton = ({ label, onClick, icon, badgeLabel, badgeFilled, isSelected, withBorder, color, hoverColor, style }: Props) => {
  return (
    <StyledButton onClick={onClick} className={isSelected ? 'selected' : 'not-selected'} $withBorder={withBorder} $color={color} $hoverColor={hoverColor} style={style}>
      {icon && <Image src={icon} alt='' width={16} height={16} />}
      <Text size={14} style={{ whiteSpace: 'nowrap' }}>
        {label}
      </Text>
      {badgeLabel !== undefined && <Badge label={badgeLabel} filled={badgeFilled || isSelected} />}
    </StyledButton>
  );
};
