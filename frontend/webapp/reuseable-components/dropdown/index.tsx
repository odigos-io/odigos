import React, { useState, useRef } from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Input } from '../input';
import { Divider } from '../divider';
import { DropdownOption } from '@/types';
import { FieldLabel } from '../field-label';
import { useOnClickOutside } from '@/hooks';
import { NoDataFound } from '../no-data-found';
import styled, { css } from 'styled-components';
import theme, { hexPercentValues } from '@/styles/theme';

interface DropdownProps {
  options: DropdownOption[];
  value: DropdownOption | undefined;
  onSelect: (option: DropdownOption) => void;
  title?: string;
  tooltip?: string;
  placeholder?: string;
  showSearch?: boolean;
  required?: boolean;
}

const RelativeContainer = styled.div`
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
`;

const DropdownHeader = styled.div<{ isOpen: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  padding: 0 16px;
  border-radius: 32px;
  cursor: pointer;

  ${({ isOpen, theme }) =>
    isOpen
      ? css`
          border: 1px solid ${theme.colors.white_opacity['40']};
          background: ${theme.colors.white_opacity['008']};
        `
      : css`
          border: 1px solid ${theme.colors.border};
          background: transparent;
        `};

  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const ArrowIcon = styled(Image)`
  &.open {
    transform: rotate(180deg);
  }
  &.close {
    transform: rotate(0deg);
  }
  transition: transform 0.3s;
`;

const AbsoluteContainer = styled.div`
  position: absolute;
  top: calc(100% + 8px);
  left: 0;
  z-index: 1;
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  width: calc(100% - 16px);
  max-height: 200px;
  gap: 8px;
  padding: 8px;
  background-color: ${({ theme }) => theme.colors.dropdown_bg_2};
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 24px;
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
    background: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['048']};
  }
  ${({ isSelected, theme }) =>
    isSelected &&
    css`
      background: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['048']};
    `}
`;

const Dropdown: React.FC<DropdownProps> = ({ options, value, onSelect, title, tooltip, placeholder, showSearch = true, required }) => {
  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = () => setIsOpen((prev) => !prev);

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, () => setIsOpen(false));

  return (
    <RelativeContainer ref={ref}>
      <FieldLabel title={title} required={required} tooltip={tooltip} style={{ marginLeft: '8px' }} />

      <DropdownHeader isOpen={isOpen} onClick={toggleOpen}>
        <Text size={14} color={!!value?.value ? undefined : theme.text.grey}>
          {value?.value || placeholder}
        </Text>
        <ArrowIcon src='/icons/common/extend-arrow.svg' alt='open-dropdown' width={14} height={14} className={isOpen ? 'open' : 'close'} />
      </DropdownHeader>

      {isOpen && (
        <DropdownContent
          options={options}
          value={value}
          onSelect={(option) => {
            onSelect(option);
            toggleOpen();
          }}
          showSearch={showSearch}
        />
      )}
    </RelativeContainer>
  );
};

export { Dropdown };

interface DropdownContentProps {
  options: DropdownProps['options'];
  value: DropdownProps['value'];
  onSelect: DropdownProps['onSelect'];
  showSearch: DropdownProps['showSearch'];
}

const DropdownContent: React.FC<DropdownContentProps> = ({ options, value, onSelect, showSearch }) => {
  const [searchText, setSearchText] = useState('');
  const filteredOptions = options.filter((option) => option.value.toLowerCase().includes(searchText.toLowerCase()));

  return (
    <AbsoluteContainer>
      {showSearch && (
        <SearchInputContainer>
          <Input placeholder='Search...' icon='/icons/common/search.svg' value={searchText} onChange={(e) => setSearchText(e.target.value.toLowerCase())} />
          <Divider thickness={1} margin='8px 0 0 0' />
        </SearchInputContainer>
      )}

      {filteredOptions.length === 0 ? (
        <NoDataFound subTitle=' ' />
      ) : (
        filteredOptions.map((opt) => {
          const isSelected = opt.id === value?.id;

          return (
            <DropdownItem key={`dropdown-option-${opt.id}`} isSelected={isSelected} onClick={() => onSelect(opt)}>
              <Text size={14}>{opt.value}</Text>
              {isSelected && <Image src='/icons/common/check.svg' alt='' width={16} height={16} />}
            </DropdownItem>
          );
        })
      )}
    </AbsoluteContainer>
  );
};
