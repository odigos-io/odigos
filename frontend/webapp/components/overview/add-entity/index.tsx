import React, { useState, useRef } from 'react';
import { PlusIcon } from '@/assets';
import { useModalStore } from '@/store';
import { getEntityIcon } from '@/utils';
import { hexPercentValues } from '@/styles';
import styled, { css, useTheme } from 'styled-components';
import { useComputePlatform, useOnClickOutside } from '@/hooks';
import { Button, FadeLoader, Text } from '@/reuseable-components';
import { type DropdownOption, OVERVIEW_ENTITY_TYPES } from '@/types';

// Styled components for the dropdown UI
const Container = styled.div`
  position: relative;
  display: inline-block;
`;

const StyledButton = styled(Button)`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 100px;
`;

const DropdownListContainer = styled.div`
  position: absolute;
  right: 0;
  top: 48px;
  border-radius: 24px;
  width: 200px;
  overflow-y: auto;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  z-index: 9999;
  padding: 12px;
`;

const DropdownItem = styled.div<{ $selected: boolean }>`
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 24px;
  gap: 8px;
  display: flex;
  align-items: center;
  &:hover {
    background: ${({ theme }) => theme.colors.dropdown_bg_2};
  }
  ${({ $selected, theme }) =>
    $selected &&
    css`
      background: ${theme.colors.majestic_blue + hexPercentValues['024']};
    `}
`;

const ButtonText = styled(Text)`
  color: ${({ theme }) => theme.text.primary};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 600;
`;

// Default options for the dropdown
const DEFAULT_OPTIONS: DropdownOption[] = [
  { id: OVERVIEW_ENTITY_TYPES.RULE, value: 'Instrumentation Rule' },
  { id: OVERVIEW_ENTITY_TYPES.SOURCE, value: 'Source' },
  { id: OVERVIEW_ENTITY_TYPES.ACTION, value: 'Action' },
  { id: OVERVIEW_ENTITY_TYPES.DESTINATION, value: 'Destination' },
];

interface Props {
  options?: DropdownOption[];
  placeholder?: string;
}

export const AddEntity: React.FC<Props> = ({ options = DEFAULT_OPTIONS, placeholder = 'ADD...' }) => {
  const theme = useTheme();
  const { loading } = useComputePlatform();
  const { currentModal, setCurrentModal } = useModalStore();

  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(dropdownRef, () => setIsDropdownOpen(false));

  const handleToggle = () => {
    setIsDropdownOpen((prev) => !prev);
  };

  const handleSelect = (option: DropdownOption) => {
    setCurrentModal(option.id);
    setIsDropdownOpen(false); // ??? maybe remove this line (for fast-toggle between modals)
  };

  return (
    <Container ref={dropdownRef}>
      <StyledButton data-id='add-entity' onClick={handleToggle}>
        {loading ? <FadeLoader color={theme.colors.primary} /> : <PlusIcon fill={theme.colors.primary} />}
        <ButtonText size={14}>{placeholder}</ButtonText>
      </StyledButton>

      {isDropdownOpen && (
        <DropdownListContainer>
          {options.map((option) => {
            const Icon = getEntityIcon(option.id as OVERVIEW_ENTITY_TYPES);

            return (
              <DropdownItem key={option.id} data-id={`add-${option.id}`} $selected={currentModal === option.id} onClick={() => handleSelect(option)}>
                <Icon />
                <Text size={14}>{option.value}</Text>
              </DropdownItem>
            );
          })}
        </DropdownListContainer>
      )}
    </Container>
  );
};
