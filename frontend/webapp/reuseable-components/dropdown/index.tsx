<<<<<<< HEAD
import React, { useState, useRef, useEffect } from 'react';
import { Input } from '../input';
import Image from 'next/image';
import { Text } from '../text';
import ReactDOM from 'react-dom';
import { Divider } from '../divider';
=======
import React, { useState, useRef } from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Badge } from '../badge';
import { Input } from '../input';
import { Divider } from '../divider';
import { Checkbox } from '../checkbox';
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
import { DropdownOption } from '@/types';
import { FieldLabel } from '../field-label';
import { useOnClickOutside } from '@/hooks';
import { NoDataFound } from '../no-data-found';
import styled, { css } from 'styled-components';
<<<<<<< HEAD

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

const Container = styled.div`
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
`;

const DropdownHeader = styled.div<{ isOpen: boolean }>`
=======
import theme, { hexPercentValues } from '@/styles/theme';

interface DropdownProps {
  title?: string;
  tooltip?: string;
  placeholder?: string;
  options: DropdownOption[];
  value: DropdownOption | DropdownOption[] | undefined;
  onSelect: (option: DropdownOption) => void;
  onDeselect: (option: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
}

const RootContainer = styled.div`
  display: flex;
  flex-direction: column;
  min-width: 120px;
  width: 100%;
`;

const RelativeContainer = styled.div`
  position: relative;
`;

const DropdownHeader = styled.div<{ isOpen: boolean; isMulti?: boolean; hasSelections?: boolean }>`
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
<<<<<<< HEAD
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
  margin-top: 12px;
  background-color: #242424;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 32px;
  z-index: 9999;
=======
  padding: ${({ isMulti, hasSelections }) => (isMulti && hasSelections ? '0 16px 0 6px' : '0 16px')};
  border-radius: 32px;
  cursor: pointer;

  ${({ isOpen, isMulti, theme }) =>
    isOpen && !isMulti
      ? css`
          border: 1px solid ${theme.colors.white_opacity['40']};
          background: ${theme.colors.white_opacity['008']};
        `
      : css`
          border: 1px solid ${theme.colors.border};
          background: transparent;
        `};

  &:hover {
    border-color: ${({ isMulti, hasSelections, theme }) => (isMulti && hasSelections ? theme.colors.border : theme.colors.secondary)};
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

export const Dropdown: React.FC<DropdownProps> = ({ options, value, onSelect, onDeselect, title, tooltip, placeholder, isMulti = false, showSearch = false, required = false }) => {
  const [isOpen, setIsOpen] = useState(false);
  const toggleOpen = () => setIsOpen((prev) => !prev);

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, () => setIsOpen(false));

  const arrLen = Array.isArray(value) ? value.length : 0;

  return (
    <RootContainer>
      <FieldLabel title={title} required={required} tooltip={tooltip} style={{ marginLeft: '8px' }} />

      <RelativeContainer ref={ref}>
        <DropdownHeader isOpen={isOpen} isMulti={isMulti} hasSelections={Array.isArray(value) ? !!value.length : false} onClick={toggleOpen}>
          <DropdownPlaceholder value={value} placeholder={placeholder} onDeselect={onDeselect} />
          <IconWrapper>
            {isMulti && <Badge label={arrLen} filled={!!arrLen} />}
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
    </RootContainer>
  );
};

const MultiLabelWrapper = styled(IconWrapper)`
  max-width: calc(100% - 50px);
  overflow-x: auto;
`;

const MultiLabel = styled(Text)`
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px 12px;
  background: ${({ theme }) => theme.colors.white_opacity['008']};
  border-radius: 360px;
  white-space: nowrap;
  text-overflow: ellipsis;
  img {
    &:hover {
      transform: scale(2);
      transition: transform 0.3s;
    }
  }
`;

const Label = styled(Text)``;

const DropdownPlaceholder: React.FC<{
  value: DropdownProps['value'];
  placeholder: DropdownProps['placeholder'];
  onDeselect: DropdownProps['onDeselect'];
}> = ({ value, placeholder, onDeselect }) => {
  if (Array.isArray(value)) {
    return !!value.length ? (
      <MultiLabelWrapper>
        {value.map((opt) => (
          <MultiLabel key={`multi-label-${opt.id}`} size={14}>
            {opt.value}
            <Divider orientation='vertical' length='10px' margin='0 4px' />
            <Image
              src='/icons/common/cross.svg'
              alt=''
              width={12}
              height={12}
              onClick={(e) => {
                e.stopPropagation();
                onDeselect?.(opt);
              }}
            />
          </MultiLabel>
        ))}
      </MultiLabelWrapper>
    ) : (
      <Label size={14} color={theme.text.grey}>
        {placeholder}
      </Label>
    );
  }

  return (
    <Label size={14} color={!!value?.value ? undefined : theme.text.grey}>
      {value?.value || placeholder}
    </Label>
  );
};

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
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
`;

const SearchInputContainer = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
`;

<<<<<<< HEAD
const DropdownItem = styled.div<{ isSelected: boolean }>`
=======
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
        <NoDataFound subTitle={showSearch && !!searchText ? undefined : ' '} />
      ) : (
        filteredOptions.map((opt) => <DropdownListItem key={`dropdown-option-${opt.id}`} option={opt} value={value} isMulti={isMulti} onSelect={onSelect} onDeselect={onDeselect} />)
      )}
    </AbsoluteContainer>
  );
};

const DropdownItem = styled.div`
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
  padding: 8px 12px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 32px;
<<<<<<< HEAD
  &:hover {
    background: rgba(68, 74, 217, 0.24);
  }
  ${({ isSelected, theme }) =>
    isSelected &&
    css`
      background: rgba(68, 74, 217, 0.24);
    `}
`;

const OpenDropdownIcon = styled(Image)<{ isOpen: boolean }>`
  transition: transform 0.3s;
  transform: ${({ isOpen }) => (isOpen ? 'rotate(180deg)' : 'rotate(0deg)')};
`;

const Dropdown: React.FC<DropdownProps> = ({ options, value, onSelect, title, tooltip, placeholder, showSearch = true, required }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [dropdownPosition, setDropdownPosition] = useState({
    top: 0,
    left: 0,
    width: 0,
  });

  const [isDisabled, setIsDisabled] = useState(false); // Disable flag for debounce
  const containerRef = useRef<HTMLDivElement>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

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

  const filteredOptions = options.filter((option) => option.value.toLowerCase().includes(searchTerm.toLowerCase()));

  const handleSelect = (option: DropdownOption) => {
    onSelect(option);
    setIsOpen(false);
  };

  const handleDropdownToggle = (e: React.MouseEvent) => {
    e.stopPropagation();

    if (isDisabled) {
      return; // Prevent multiple clicks if debounce is active
    }

    // Toggle dropdown open/close state
    setIsOpen((prev) => !prev);

    // Set the disable flag to true and reset after 1 second
    setIsDisabled(true);
    setTimeout(() => {
      setIsDisabled(false);
    }, 1000); // 1 second debounce delay
  };

  const dropdownContent = (
    <div
      style={{
        position: 'absolute',
        top: dropdownPosition.top,
        left: dropdownPosition.left,
        width: dropdownPosition.width,
      }}
      onClick={(e) => e.stopPropagation()}
    >
      <DropdownListContainer ref={dropdownRef}>
        {showSearch && (
          <SearchInputContainer>
            <Input placeholder='Search...' icon={'/icons/common/search.svg'} value={searchTerm} onChange={(e) => setSearchTerm(e.target.value)} />
            <Divider thickness={1} margin='8px 0 0 0' />
          </SearchInputContainer>
        )}
        {filteredOptions.length === 0 && <NoDataFound title='No data found' subTitle=' ' />}
        {filteredOptions.map((option) => (
          <DropdownItem key={option.id} isSelected={option.id === value?.id} onClick={() => handleSelect(option)}>
            <Text size={14}>{option.value}</Text>
            {option.id === value?.id && <Image src='/icons/common/check.svg' alt='' width={16} height={16} />}
          </DropdownItem>
        ))}
      </DropdownListContainer>
    </div>
  );

  return (
    <Container ref={containerRef}>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      <DropdownHeader isOpen={isOpen} onClick={handleDropdownToggle}>
        <Text size={14}>{value?.value || placeholder}</Text>
        <OpenDropdownIcon src='/icons/common/extend-arrow.svg' alt='open-dropdown' width={12} height={12} isOpen={isOpen} />
      </DropdownHeader>

      {isOpen && ReactDOM.createPortal(dropdownContent, document.body)}
    </Container>
  );
};

export { Dropdown };
=======
  &:hover,
  &.selected {
    background: ${({ theme }) => theme.colors.majestic_blue + hexPercentValues['024']};
  }
`;

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
      <DropdownItem className={isSelected ? 'selected' : ''}>
        <Checkbox title={option.value} titleColor={theme.text.secondary} initialValue={isSelected} onChange={(toAdd) => (toAdd ? onSelect(option) : onDeselect?.(option))} style={{ width: '100%' }} />
      </DropdownItem>
    );
  }

  return (
    <DropdownItem className={isSelected ? 'selected' : ''} onClick={() => (isSelected ? onDeselect?.(option) : onSelect(option))}>
      <Text size={14}>{option.value}</Text>
      {isSelected && <Image src='/icons/common/check.svg' alt='' width={16} height={16} />}
    </DropdownItem>
  );
};
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
