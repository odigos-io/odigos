import React, { useState, useRef } from 'react';
import { useOnClickOutside } from '@/hooks';
import type { DropdownOption } from '@/types';
import styled, { css } from 'styled-components';
import theme, { hexPercentValues } from '@/styles/theme';
import { CheckIcon, CrossIcon, SearchIcon } from '@/assets';
import { Badge, Checkbox, Divider, ExtendIcon, FieldError, FieldLabel, Input, NoDataFound, Text } from '@/reuseable-components';

interface DropdownProps {
  title?: string;
  tooltip?: string;
  placeholder?: string;
  options: DropdownOption[];
  value: DropdownOption | DropdownOption[] | undefined;
  onSelect: (option: DropdownOption) => void;
  onDeselect?: (option: DropdownOption) => void;
  isMulti?: boolean;
  required?: boolean;
  showSearch?: boolean;
  errorMessage?: string;
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

const DropdownHeader = styled.div<{ $isOpen: boolean; $isMulti?: boolean; $hasSelections: boolean; $hasError: boolean }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 36px;
  padding: ${({ $isMulti, $hasSelections }) => ($isMulti && $hasSelections ? '0 16px 0 6px' : '0 16px')};
  border-radius: 32px;
  cursor: pointer;

  ${({ $isOpen, $isMulti, theme }) =>
    $isOpen && !$isMulti
      ? css`
          border: 1px solid ${theme.colors.white_opacity['40']};
          background: ${theme.colors.white_opacity['008']};
        `
      : css`
          border: 1px solid ${theme.colors.border};
          background: transparent;
        `};

  ${({ $hasError }) =>
    $hasError &&
    css`
      border-color: ${({ theme }) => theme.text.error};
    `}

  &:hover {
    border-color: ${({ $isMulti, $hasSelections, theme }) => ($isMulti && $hasSelections ? theme.colors.border : theme.colors.secondary)};
  }
`;

const IconWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 4px;
`;

export const Dropdown: React.FC<DropdownProps> = ({ options, value, onSelect, onDeselect, title, tooltip, placeholder, isMulti = false, showSearch = false, required = false, errorMessage }) => {
  const [isOpen, setIsOpen] = useState(false);
  const [openUpwards, setOpenUpwards] = useState(false);

  const ref = useRef<HTMLDivElement>(null);
  useOnClickOutside(ref, () => setIsOpen(false));

  const handleDirection = () => {
    if (ref.current) {
      const rect = ref.current.getBoundingClientRect();
      const isNearBottom = rect.bottom + 300 > window.innerHeight;
      setOpenUpwards(isNearBottom);
    }
  };

  const toggleOpen = () => {
    handleDirection();
    setIsOpen((prev) => !prev);
  };

  const arrLen = Array.isArray(value) ? value.length : 0;

  return (
    <RootContainer>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      <RelativeContainer ref={ref}>
        <DropdownHeader $isOpen={isOpen} $isMulti={isMulti} $hasSelections={Array.isArray(value) ? !!value.length : false} $hasError={!!errorMessage} onClick={toggleOpen}>
          <DropdownPlaceholder value={value} placeholder={placeholder} onDeselect={onDeselect} />
          <IconWrapper>
            {isMulti && <Badge label={arrLen} filled={!!arrLen} />}
            <ExtendIcon extend={isOpen} />
          </IconWrapper>
        </DropdownHeader>

        {isOpen && (
          <DropdownList
            openUpwards={openUpwards}
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

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
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
            <CrossIcon
              size={12}
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

const AbsoluteContainer = styled.div<{ $openUpwards: boolean }>`
  position: absolute;
  ${({ $openUpwards }) => ($openUpwards ? 'bottom' : 'top')}: calc(100% + 8px);
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

const DropdownList: React.FC<{
  openUpwards: boolean;
  options: DropdownProps['options'];
  value: DropdownProps['value'];
  onSelect: DropdownProps['onSelect'];
  onDeselect: DropdownProps['onDeselect'];
  isMulti: DropdownProps['isMulti'];
  showSearch: DropdownProps['showSearch'];
}> = ({ openUpwards, options, value, onSelect, onDeselect, isMulti, showSearch }) => {
  const [searchText, setSearchText] = useState('');
  const filteredOptions = options.filter((option) => option.value.toLowerCase().includes(searchText));

  return (
    <AbsoluteContainer $openUpwards={openUpwards}>
      {showSearch && (
        <SearchInputContainer>
          <Input placeholder='Search...' icon={SearchIcon} value={searchText} onChange={(e) => setSearchText(e.target.value.toLowerCase())} />
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
  padding: 8px 12px;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-radius: 32px;
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
        <Checkbox title={option.value} titleColor={theme.text.secondary} value={isSelected} onChange={(toAdd) => (toAdd ? onSelect(option) : onDeselect?.(option))} style={{ width: '100%' }} />
      </DropdownItem>
    );
  }

  return (
    <DropdownItem className={isSelected ? 'selected' : ''} onClick={() => (isSelected ? onDeselect?.(option) : onSelect(option))}>
      <Text size={14}>{option.value}</Text>
      {isSelected && <CheckIcon />}
    </DropdownItem>
  );
};
