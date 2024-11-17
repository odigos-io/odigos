import React from 'react';
import styled, { css } from 'styled-components';

interface TagProps {
  id: string;
  children: React.ReactNode;
  isSelected: boolean;
  isDisabled?: boolean;
  onClick: (id: string) => void;
}

const TagContainer = styled.div<{ $selected: boolean; $disabled: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  height: 36px;
  gap: 6px;
  padding: 0 12px;
  border-radius: 32px;
  background-color: ${({ theme, $selected }) => ($selected ? theme.colors.primary : theme.colors.translucent_bg)};
  cursor: ${({ $disabled }) => ($disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ $disabled }) => ($disabled ? 0.5 : 1)};
  transition: background-color 0.2s ease-in-out, color 0.2s ease-in-out;

  ${({ $disabled, theme }) =>
    !$disabled &&
    css`
      &:hover {
        background-color: ${theme.colors.primary};
      }
    `}
`;

const Tag: React.FC<TagProps> = ({ id, isSelected, isDisabled = false, onClick, children }) => {
  const handleClick = () => {
    if (!isDisabled) onClick(id);
  };

  return (
    <TagContainer $selected={isSelected} $disabled={isDisabled} onClick={handleClick} role='button' aria-disabled={isDisabled} aria-pressed={isSelected}>
      {children}
    </TagContainer>
  );
};

export { Tag };
