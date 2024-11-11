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
import { Badge } from '../badge';
import { Checkbox } from '../checkbox';

interface DropdownProps {
  options: DropdownOption[];
  value?: DropdownOption | DropdownOption[];
  onSelect: (option: DropdownOption) => void;
  onDeselect?: (option: DropdownOption) => void;
  title?: string;
  tooltip?: string;
  placeholder?: string;
  isMulti?: boolean;
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

const IconWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 4px;
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
    background: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['024']};
  }
  ${({ isSelected, theme }) =>
    isSelected &&
    css`
      background: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['024']};
    `}
`;

export const Dropdown: React.FC<DropdownProps> = ({ options, value, onSelect, onDeselect, title, tooltip, placeholder, isMulti = false, showSearch = true, required = false }) => {
  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = () => setIsOpen((prev) => !prev);

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, () => setIsOpen(false));

  return (
    <RelativeContainer ref={ref}>
      <FieldLabel title={title} required={required} tooltip={tooltip} style={{ marginLeft: '8px' }} />

      <DropdownHeader isOpen={isOpen} onClick={toggleOpen}>
        <DropdownPlaceholder value={value} placeholder={placeholder} />

        <IconWrapper>
          {isMulti && <Badge label={(value as DropdownOption[]).length} filled />}
          <ArrowIcon src='/icons/common/extend-arrow.svg' alt='open-dropdown' width={14} height={14} className={isOpen ? 'open' : 'close'} />
        </IconWrapper>
      </DropdownHeader>

      {isOpen && (
        <DropdownList
          options={options}
          value={value}
          onSelect={(option) => {
            onSelect(option);
            if (!isMulti) toggleOpen();
          }}
          onDeselect={(option) => {
            onDeselect?.(option);
            if (!isMulti) toggleOpen();
          }}
          isMulti={isMulti}
          showSearch={showSearch}
        />
      )}
    </RelativeContainer>
  );
};

const DropdownPlaceholder: React.FC<{
  value: DropdownProps['value'];
  placeholder: DropdownProps['placeholder'];
}> = ({ value, placeholder }) => {
  if (Array.isArray(value)) {
    return !!value.length ? (
      <Text size={14} color={undefined}>
        {value.map((s) => s.value).join(', ')}
      </Text>
    ) : (
      <Text size={14} color={theme.text.grey}>
        {placeholder}
      </Text>
    );
  }

  return (
    <Text size={14} color={!!value?.value ? undefined : theme.text.grey}>
      {value?.value || placeholder}
    </Text>
  );
};

const DropdownList: React.FC<{
  options: DropdownProps['options'];
  value: DropdownProps['value'];
  onSelect: DropdownProps['onSelect'];
  onDeselect: DropdownProps['onDeselect'];
  isMulti: DropdownProps['isMulti'];
  showSearch: DropdownProps['showSearch'];
}> = ({ options, value, onSelect, onDeselect, isMulti, showSearch }) => {
  const [searchText, setSearchText] = useState('');
  const filteredOptions = options.filter((option) => option.value.toLowerCase().includes(searchText));

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
        filteredOptions.map((opt) => <DropdownListItem key={`dropdown-option-${opt.id}`} option={opt} value={value} isMulti={isMulti} onSelect={onSelect} onDeselect={onDeselect} />)
      )}
    </AbsoluteContainer>
  );
};

const DropdownListItem: React.FC<{
  option: DropdownOption;
  value: DropdownProps['value'];
  isMulti: DropdownProps['isMulti'];
  onSelect: DropdownProps['onSelect'];
  onDeselect: DropdownProps['onDeselect'];
}> = ({ option, value, isMulti, onSelect, onDeselect }) => {
  const isSelected = Array.isArray(value) ? !!value?.find((s) => s.id === option.id) : value?.id === option.id;

  if (isMulti) {
    return (
      <DropdownItem isSelected={isSelected}>
        <Checkbox title={option.value} titleColor={theme.text.secondary} initialValue={isSelected} onChange={(toAdd) => (toAdd ? onSelect(option) : onDeselect?.(option))} style={{ width: '100%' }} />
      </DropdownItem>
    );
  }

  return (
    <DropdownItem isSelected={isSelected} onClick={() => (isSelected ? onDeselect?.(option) : onSelect(option))}>
      <Text size={14}>{option.value}</Text>
      {isSelected && <Image src='/icons/common/check.svg' alt='' width={16} height={16} />}
    </DropdownItem>
  );
};
