import React, { useState, ChangeEvent, KeyboardEvent, FC } from 'react';
import styled, { css } from 'styled-components';
import { Text } from '../text';
import Image from 'next/image';

interface Option {
  id: string;
  label: string;
  description?: string;
  icon?: string;
  items?: Option[]; // For handling a list of items
}

interface AutocompleteInputProps {
  options: Option[];
  placeholder?: string;
  onOptionSelect?: (option: Option) => void;
}

const AutocompleteInput: FC<AutocompleteInputProps> = ({
  options,
  placeholder = 'Type to search...',
  onOptionSelect,
}) => {
  const [query, setQuery] = useState('');
  const [filteredOptions, setFilteredOptions] = useState<Option[]>([]);
  const [showOptions, setShowOptions] = useState(false);
  const [activeIndex, setActiveIndex] = useState(-1);

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value;
    setQuery(input);
    if (input) {
      const filtered = filterOptions(options, input);
      setFilteredOptions(filtered);
      setShowOptions(true);
    } else {
      setShowOptions(false);
    }
  };

  const filterOptions = (optionsList: Option[], input: string): Option[] => {
    return optionsList.reduce<Option[]>((acc, option) => {
      if (option.items) {
        const filteredSubItems = filterOptions(option.items, input);
        if (filteredSubItems.length) {
          acc.push({ ...option, items: filteredSubItems });
        }
      } else if (option.label.toLowerCase().includes(input.toLowerCase())) {
        acc.push(option);
      }
      return acc;
    }, []);
  };

  const handleOptionClick = (option: Option) => {
    setQuery(option.label);
    setShowOptions(false);
    if (onOptionSelect) {
      onOptionSelect(option);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown' && activeIndex < filteredOptions.length - 1) {
      setActiveIndex(activeIndex + 1);
    } else if (e.key === 'ArrowUp' && activeIndex > 0) {
      setActiveIndex(activeIndex - 1);
    } else if (e.key === 'Enter' && activeIndex >= 0) {
      handleOptionClick(filteredOptions[activeIndex]);
    }
  };

  return (
    <AutocompleteContainer>
      <InputWrapper>
        <StyledInput
          type="text"
          value={query}
          placeholder={placeholder}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onBlur={() => setShowOptions(false)}
          onFocus={() => query && setShowOptions(true)}
        />
      </InputWrapper>
      {showOptions && (
        <OptionsList>
          {filteredOptions.map((option, index) => (
            <OptionItem
              key={option.id}
              option={option}
              isActive={index === activeIndex}
              onClick={() => handleOptionClick(option)}
            />
          ))}
        </OptionsList>
      )}
    </AutocompleteContainer>
  );
};

interface OptionItemProps {
  option: Option;
  isActive: boolean;
  onClick: () => void;
}

const OptionItem: FC<OptionItemProps> = ({ option, isActive, onClick }) => {
  return (
    <OptionItemContainer
      isActive={isActive}
      isList={!!option.items && option.items.length > 0}
      onMouseDown={onClick}
    >
      {option.icon && (
        <Image width={16} height={16} src={option.icon} alt={option.label} />
      )}
      <div>
        <OptionLabelWrapper>
          <OptionLabel>{option.label}</OptionLabel>
          <OptionDescription>{option.description}</OptionDescription>
        </OptionLabelWrapper>
        {option.items && option.items.length > 0 && (
          <SubOptionsList>
            {option.items.map((subOption) => (
              <OptionItem
                key={subOption.id}
                option={subOption}
                isActive={false}
                onClick={() => onClick()}
              />
            ))}
          </SubOptionsList>
        )}
      </div>
    </OptionItemContainer>
  );
};

export { AutocompleteInput };

/** Styled Components */

const AutocompleteContainer = styled.div`
  position: relative;
`;

const InputWrapper = styled.div<{}>`
  width: calc(100% - 16px);
  display: flex;
  align-items: center;
  height: 36px;
  gap: 12px;
  padding: 0 8px;
  transition: border-color 0.3s;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);

  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const StyledInput = styled.input`
  flex: 1;
  border: none;
  outline: none;
  background: none;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-weight: 300;
  &::placeholder {
    color: ${({ theme }) => theme.colors.text};
    font-family: ${({ theme }) => theme.font_family.primary};
    opacity: 0.4;
    font-size: 14px;
    font-weight: 300;
    line-height: 22px; /* 157.143% */
  }

  &:disabled {
    background-color: #555;
    cursor: not-allowed;
  }
`;

const OptionsList = styled.ul`
  position: absolute;
  max-height: 348px;
  top: 32px;
  border-radius: 24px;
  width: calc(100% - 16px);
  overflow-y: auto;
  background-color: ${({ theme }) => theme.colors.dropdown_bg};
  border: 1px solid ${({ theme }) => theme.colors.border};
  z-index: 9999;
  padding: 12px;
`;

interface OptionItemContainerProps {
  isActive: boolean;
  isList: boolean;
}

const OptionItemContainer = styled.li<OptionItemContainerProps>`
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 24px;
  gap: 8px;
  display: flex;
  align-items: ${({ isList }) => (isList ? 'flex-start' : 'center')};
  &:hover {
    background: ${({ theme }) => theme.colors.white_opacity['008']};
  }
`;

const OptionLabelWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

const OptionLabel = styled(Text)`
  flex: 1;
  font-size: 14px;
`;

const OptionDescription = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 150%;
`;
const SubOptionsList = styled.ul`
  padding-left: 16px;
  margin: 4px 0 0 0;
  list-style: none;
`;
