import React, { useState, useRef } from 'react';
import Image from 'next/image';
import { DropdownOption, K8sActualSource } from '@/types';
import { useConnectSourcesMenuState, useOnClickOutside } from '@/hooks';
import styled, { css } from 'styled-components';
import { Button, Modal, Text } from '@/reuseable-components';
import { ChooseSourcesContainer } from '../../sources';
import { useAppStore } from '@/store';
import { ChooseSourcesBody } from '../../sources/choose-sources/choose-sources-body';

interface AddEntityButtonDropdownProps {
  options?: DropdownOption[];
  onSelect: (option: DropdownOption) => void;
  placeholder?: string;
}

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
  right: 0px;
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

const OPTIONS = [
  {
    id: 'sources',
    value: 'Source',
  },
  {
    id: 'actions',
    value: 'Action',
  },
  {
    id: 'destinations',
    value: 'Destination',
  },
];

const AddEntityButtonDropdown: React.FC<AddEntityButtonDropdownProps> = ({
  options = OPTIONS,
  onSelect,
  placeholder = 'ADD...',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const [sourcesList, setSourcesList] = useState<K8sActualSource[]>([]);

  const { setSources, setNamespaceFutureSelectAppsList } = useAppStore();
  const { stateMenu, stateHandlers } = useConnectSourcesMenuState({
    sourcesList,
  });

  useOnClickOutside(dropdownRef, () => setIsOpen(false));

  const handleToggle = () => {
    setIsOpen((prev) => !prev);
  };

  const handleSelect = (option: DropdownOption) => {
    onSelect(option);
    setIsOpen(false);
  };

  return (
    <Container ref={dropdownRef}>
      <StyledButton onClick={handleToggle}>
        <Image
          src="/icons/common/plus-black.svg"
          width={16}
          height={16}
          alt="Add"
        />
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
                alt="Add"
              />
              <Text size={14}>{option.value}</Text>
            </DropdownItem>
          ))}
        </DropdownListContainer>
      )}
      <Modal isOpen={true} header={{ title: 'ADD SOURCE' }} onClose={() => {}}>
        <div
          style={{
            width: '1080px',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            flexDirection: 'column',
          }}
        >
          <ChooseSourcesBody
            stateMenu={stateMenu}
            stateHandlers={stateHandlers}
            sourcesList={sourcesList}
            setSourcesList={setSourcesList}
          />
        </div>
      </Modal>
    </Container>
  );
};

export { AddEntityButtonDropdown };
