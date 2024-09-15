import React, { useState, useRef, useEffect } from 'react';
import { Input } from '../input';
import styled, { css } from 'styled-components';
import { Tooltip } from '../tooltip';
import Image from 'next/image';
import { Text } from '../text';
import { Divider } from '../divider';
import { DropdownOption } from '@/types';
import { useOnClickOutside } from '@/hooks';
import ReactDOM from 'react-dom';

interface DropdownProps {
  options: DropdownOption[];
  value: DropdownOption | undefined;
  onSelect: (option: DropdownOption) => void;
  title?: string;
  tooltip?: string;
  placeholder?: string;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
`;

const Title = styled(Text)``;

const DropdownHeader = styled.div<{ isOpen: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  padding: 0 16px;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);
  cursor: pointer;
  background-color: transparent;
  ${({ isOpen, theme }) =>
    isOpen &&
    css`
      border: 1px solid rgba(249, 249, 249, 0.48);
      background: rgba(249, 249, 249, 0.08);
    `};

  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const DropdownListContainer = styled.div`
  position: absolute;
  width: 100%;
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
  background-color: #242424;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 32px;
  z-index: 9999;
`;

const SearchInputContainer = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
`;

const DropdownItem = styled.div<{ isSelected: boolean }>`
  padding: 8px 12px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 32px;
  &:hover {
    background: rgba(68, 74, 217, 0.24);
  }
  ${({ isSelected, theme }) =>
    isSelected &&
    css`
      background: rgba(68, 74, 217, 0.24);
    `}
`;

const HeaderWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
`;

const OpenDropdownIcon = styled(Image)<{ isOpen: boolean }>`
  transform: ${({ isOpen }) => (isOpen ? 'rotate(180deg)' : 'rotate(0deg)')};
`;

const Dropdown: React.FC<DropdownProps> = ({
  options,
  value,
  onSelect,
  title,
  tooltip,
  placeholder,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [dropdownPosition, setDropdownPosition] = useState({
    top: 0,
    left: 0,
    width: 0,
  });
  const dropdownRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useOnClickOutside(dropdownRef, () => setIsOpen(false));

  useEffect(() => {
    if (isOpen && containerRef.current) {
      const rect = containerRef.current.getBoundingClientRect();
      setDropdownPosition({
        top: rect.bottom + window.scrollY, // Ensure correct vertical position
        left: rect.left + window.scrollX, // Ensure correct horizontal position
        width: rect.width, // Ensure dropdown matches the width of the input
      });
    }
  }, [isOpen]);

  const filteredOptions = options.filter((option) =>
    option.value.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const handleSelect = (option: DropdownOption) => {
    onSelect(option);
    setIsOpen(false);
  };

  const dropdownContent = (
    <div
      style={{
        position: 'absolute',
        top: dropdownPosition.top,
        left: dropdownPosition.left,
        width: dropdownPosition.width,
      }}
    >
      <DropdownListContainer>
        <SearchInputContainer>
          <Input
            placeholder="Search..."
            icon={'/icons/common/search.svg'}
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
          <Divider thickness={1} margin="8px 0 0 0" />
        </SearchInputContainer>
        {filteredOptions.map((option) => (
          <DropdownItem
            key={option.id}
            isSelected={option.id === value?.id}
            onClick={() => handleSelect(option)}
          >
            <Text size={14}>{option.value}</Text>
            {option.id === value?.id && (
              <Image
                src="/icons/common/check.svg"
                alt=""
                width={16}
                height={16}
              />
            )}
          </DropdownItem>
        ))}
      </DropdownListContainer>
    </div>
  );

  return (
    <Container ref={containerRef}>
      {title && (
        <Tooltip text={tooltip || ''}>
          <HeaderWrapper>
            <Title>{title}</Title>
            {tooltip && (
              <Image
                src="/icons/common/info.svg"
                alt=""
                width={16}
                height={16}
              />
            )}
          </HeaderWrapper>
        </Tooltip>
      )}
      <DropdownHeader isOpen={isOpen} onClick={() => setIsOpen(!isOpen)}>
        <Text size={14}>{value?.value || placeholder}</Text>
        <OpenDropdownIcon
          src="/icons/common/extend-arrow.svg"
          alt="open-dropdown"
          width={12}
          height={12}
          isOpen={isOpen}
        />
      </DropdownHeader>
      {isOpen && ReactDOM.createPortal(dropdownContent, document.body)}
    </Container>
  );
};

export { Dropdown };