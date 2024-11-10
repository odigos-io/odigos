import React from 'react';
import { Button } from '..';
import Image from 'next/image';
import styled from 'styled-components';
import { Badge, Text } from '@/reuseable-components';

interface Props {
  label: string;
  badgeLabel?: string | number;
  icon?: string;
  isSelected?: boolean;
  onClick: () => void;
  color?: React.CSSProperties['backgroundColor'];
  hoverColor?: React.CSSProperties['backgroundColor'];
  style?: React.CSSProperties;
}

const StyledButton = styled(Button)<{ color: Props['color']; hoverColor: Props['hoverColor'] }>`
  gap: 8px;
  text-transform: none;
  text-decoration: none;
  border: none;
  background-color: ${({ theme, color }) => color || theme.colors.white_opacity['004']};
  &.not-selected {
    &:hover {
      background-color: ${({ theme, hoverColor }) => hoverColor || theme.colors.white_opacity['008']};
    }
  }
  &.selected {
    background-color: ${({ theme }) => theme.colors.majestic_blue + '7A'};
  }
`;

export const SelectionButton = ({ label, badgeLabel, icon, isSelected, onClick, color, hoverColor, style }: Props) => {
  return (
    <StyledButton onClick={onClick} className={isSelected ? 'selected' : 'not-selected'} color={color} hoverColor={hoverColor} style={style}>
      {icon && <Image src={icon} alt='' width={16} height={16} />}
      <Text size={14} style={{ whiteSpace: 'nowrap' }}>
        {label}
      </Text>
      {badgeLabel !== undefined && <Badge label={badgeLabel} filled={isSelected} />}
    </StyledButton>
  );
};
