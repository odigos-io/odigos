import React, { useState, useRef, useCallback } from 'react';
import Image from 'next/image';
import theme from '@/styles/theme';
import { DropdownOption } from '@/types';
import styled, { css } from 'styled-components';
import { useActualSources, useOnClickOutside } from '@/hooks';
import { Button, FadeLoader, Text } from '@/reuseable-components';
import { AddSourceModal } from '../../sources/choose-sources/choose-source-modal';
import { AddDestinationModal } from '../../destinations/add-destination/add-destination-modal';
import { AddActionModal } from '../../actions';

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
  width: 131px;
  overflow-y: auto;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  z-index: 9999;
  padding: 12px;
`;

const DropdownItem = styled.div<{ isSelected: boolean }>`
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 24px;
  gap: 8px;
  display: flex;
  align-items: center;
  &:hover {
    background: ${({ theme }) => theme.colors.white_opacity['008']};
  }
  ${({ isSelected }) =>
    isSelected &&
    css`
      background: rgba(68, 74, 217, 0.24);
    `}
`;

const ButtonText = styled(Text)`
  color: ${({ theme }) => theme.text.primary};
  font-family: ${({ theme }) => theme.font_family.secondary};
  font-weight: 600;
`;

// Default options for the dropdown
const DEFAULT_OPTIONS: DropdownOption[] = [
  { id: 'sources', value: 'Source' },
  { id: 'actions', value: 'Action' },
  { id: 'destinations', value: 'Destination' },
];

interface AddEntityButtonDropdownProps {
  options?: DropdownOption[];
  placeholder?: string;
}

const AddEntityButtonDropdown: React.FC<AddEntityButtonDropdownProps> = ({
  options = DEFAULT_OPTIONS,
  placeholder = 'ADD...',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [currentModal, setCurrentModal] = useState<string>('');

  const { isPolling, createSourcesForNamespace, persistNamespaceItems } =
    useActualSources();

  useOnClickOutside(dropdownRef, () => setIsOpen(false));

  const handleToggle = useCallback(() => {
    setIsOpen((prev) => !prev);
  }, []);

  const handleSelect = useCallback((option: DropdownOption) => {
    setCurrentModal(option.id);
    setIsOpen(false);
  }, []);

  const handleCloseModal = useCallback(() => {
    setCurrentModal('');
  }, []);

  return (
    <Container ref={dropdownRef}>
      <StyledButton onClick={handleToggle}>
        {isPolling ? (
          <FadeLoader color={theme.colors.primary} />
        ) : (
          <Image
            src="/icons/common/plus-black.svg"
            width={16}
            height={16}
            alt="Add"
          />
        )}
        <ButtonText size={14}>{placeholder}</ButtonText>
      </StyledButton>
      {isOpen && (
        <DropdownListContainer>
          {options.map((option) => (
            <DropdownItem
              key={option.id}
              isSelected={false}
              onClick={() => handleSelect(option)}
            >
              <Image
                src={`/icons/overview/${option.id}.svg`}
                width={16}
                height={16}
                alt={`Add ${option.value}`}
              />
              <Text size={14}>{option.value}</Text>
            </DropdownItem>
          ))}
        </DropdownListContainer>
      )}
      <AddSourceModal
        isOpen={currentModal === 'sources'}
        onClose={handleCloseModal}
        createSourcesForNamespace={createSourcesForNamespace}
        persistNamespaceItems={persistNamespaceItems}
      />
      <AddDestinationModal
        isModalOpen={currentModal === 'destinations'}
        handleCloseModal={handleCloseModal}
      />
      <AddActionModal
        isModalOpen={currentModal === 'actions'}
        handleCloseModal={handleCloseModal}
      />
    </Container>
  );
};

export { AddEntityButtonDropdown };
