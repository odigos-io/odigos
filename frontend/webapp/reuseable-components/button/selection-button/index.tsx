import styled from 'styled-components';
import { Button } from '..';
import { Badge, Text } from '@/reuseable-components';

const StyledButton = styled(Button)`
  padding: 0 16px;
  gap: 6px;
  text-transform: none;
  text-decoration: none;
  border-color: transparent;
  background-color: ${({ theme }) => theme.colors.white_opacity['004']};
  &.selected {
    background-color: ${({ theme }) => theme.colors.majestic_blue + '7A'};
    &:hover {
      border-color: ${({ theme }) => theme.colors.majestic_blue};
    }
  }
  &.not-selected {
    &:hover {
      border-color: ${({ theme }) => theme.colors.border};
      background-color: ${({ theme }) => theme.colors.white_opacity['008']};
    }
  }
`;

interface Props {
  label: string;
  badgeLabel?: string | number;
  isSelected: boolean;
  onClick: () => void;
}

export const SelectionButton = ({ label, badgeLabel, isSelected, onClick }: Props) => {
  return (
    <StyledButton onClick={onClick} className={isSelected ? 'selected' : 'not-selected'}>
      <Text size={14} style={{ whiteSpace: 'nowrap' }}>
        {label}
      </Text>
      {badgeLabel !== undefined && <Badge label={badgeLabel} filled={isSelected} />}
    </StyledButton>
  );
};
