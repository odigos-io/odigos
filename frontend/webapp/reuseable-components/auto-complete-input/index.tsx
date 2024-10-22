import React, { useState, ChangeEvent, KeyboardEvent, FC } from 'react'
import styled from 'styled-components'
import Image from 'next/image'
import { Text } from '../text'

export interface Option {
  id: string
  label: string
  description?: string
  icon?: string
  items?: Option[] // For handling a list of items
}

interface AutocompleteInputProps {
  options: Option[]
  placeholder?: string
  onOptionSelect?: (option: Option) => void
}

const filterOptions = (optionsList: Option[], input: string): Option[] => {
  return optionsList.reduce<Option[]>((acc, option) => {
    if (option.items) {
      const filteredSubItems = filterOptions(option.items, input)
      if (filteredSubItems.length) {
        acc.push({ ...option, items: filteredSubItems })
      }
    } else if (option.label.toLowerCase().includes(input.toLowerCase())) {
      acc.push(option)
    }
    return acc
  }, [])
}

const AutocompleteInput: FC<AutocompleteInputProps> = ({ options, placeholder = 'Type to search...', onOptionSelect }) => {
  const [query, setQuery] = useState('')
  const [filteredOptions, setFilteredOptions] = useState<Option[]>(filterOptions(options, ''))
  const [showOptions, setShowOptions] = useState(false)
  const [activeIndex, setActiveIndex] = useState(-1)

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const input = e.target.value
    setQuery(input)
    if (input) {
      const filtered = filterOptions(options, input)
      setFilteredOptions(filtered)
      setShowOptions(true)
    } else {
      setShowOptions(false)
    }
  }

  const handleOptionClick = (option: Option) => {
    setQuery(option.label)
    setShowOptions(false)
    if (onOptionSelect) {
      onOptionSelect(option)
    }
  }

  const flattenOptions = (options: Option[]): Option[] => {
    return options.reduce<Option[]>((acc, option) => {
      acc.push(option)
      if (option.items) {
        acc = acc.concat(flattenOptions(option.items))
      }
      return acc
    }, [])
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    // Flatten the options to handle keyboard navigation - TODO: Refactor this
    return
    const flatOptions = flattenOptions(filteredOptions)
    if (e.key === 'ArrowDown' && activeIndex < flatOptions.length - 1) {
      setActiveIndex(activeIndex + 1)
    } else if (e.key === 'ArrowUp' && activeIndex > 0) {
      setActiveIndex(activeIndex - 1)
    } else if (e.key === 'Enter' && activeIndex >= 0) {
      handleOptionClick(flatOptions[activeIndex])
    }
  }

  return (
    <AutocompleteContainer>
      <InputWrapper>
        <StyledInput
          type='text'
          value={query}
          placeholder={placeholder}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          onBlur={() => setShowOptions(false)}
          onFocus={() => setShowOptions(true)}
        />
      </InputWrapper>
      {showOptions && (
        <OptionsList>
          {filteredOptions.map((option, index) => (
            <OptionItem key={option.id} option={option} isActive={index === activeIndex} onClick={handleOptionClick} />
          ))}
        </OptionsList>
      )}
    </AutocompleteContainer>
  )
}

interface OptionItemProps {
  option: Option
  isActive: boolean
  onClick: (option: Option) => void
}

const OptionItem: FC<OptionItemProps> = ({ option, isActive, onClick }) => {
  const hasSubItems = !!option.items && option.items.length > 0

  return (
    <OptionItemContainer isActive={isActive} isList={hasSubItems} onMouseDown={() => (hasSubItems ? null : onClick(option))}>
      {option.icon && <Image width={16} height={16} src={option.icon} alt={option.label} />}

      <OptionContent>
        <OptionLabelWrapper>
          <OptionLabel>{option.label}</OptionLabel>
          <OptionDescription>{option.description}</OptionDescription>
        </OptionLabelWrapper>

        {hasSubItems && (
          <SubOptionsList>
            {option.items?.map((subOption) => (
              <SubOptionContainer key={subOption.id}>
                <VerticalLine />
                <OptionItem option={subOption} isActive={false} onClick={onClick} />
              </SubOptionContainer>
            ))}
          </SubOptionsList>
        )}
      </OptionContent>
    </OptionItemContainer>
  )
}

export { AutocompleteInput }

/** Styled Components */

const AutocompleteContainer = styled.div`
  position: relative;
`

const InputWrapper = styled.div`
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
`

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
    line-height: 22px;
  }

  &:disabled {
    background-color: #555;
    cursor: not-allowed;
  }
`

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
`

interface OptionItemContainerProps {
  isActive: boolean
  isList: boolean
}

const OptionItemContainer = styled.li<OptionItemContainerProps>`
  width: 100%;
  padding: 8px 12px;
  cursor: ${({ isList }) => (isList ? 'default' : 'pointer')};
  border-radius: 24px;
  gap: 8px;
  display: flex;
  align-items: ${({ isList }) => (isList ? 'flex-start' : 'center')};
  background: ${({ isActive, theme }) => (isActive ? theme.colors.activeBackground : 'transparent')};

  &:hover {
    background: ${({ theme, isList }) => !isList && theme.colors.white_opacity['008']};
  }
`

const OptionContent = styled.div`
  width: 100%;
`

const SubOptionContainer = styled.div`
  display: flex;
  width: 100%;
`

const VerticalLine = styled.div`
  width: 1px;
  height: 52px;
  background-color: ${({ theme }) => theme.colors.white_opacity['008']};
  position: absolute;
  left: 33px;
`

const OptionLabelWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
`

const OptionLabel = styled(Text)`
  flex: 1;
  font-size: 14px;
`

const OptionDescription = styled(Text)`
  color: ${({ theme }) => theme.text.grey};
  font-size: 10px;
  line-height: 150%;
`

const SubOptionsList = styled.ul`
  padding-left: 0px;
  margin: 4px 0 0 0;
  list-style: none;
  width: 100%;
`
