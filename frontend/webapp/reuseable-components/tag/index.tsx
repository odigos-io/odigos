import React from 'react';
import styled, { css } from 'styled-components';

interface TagProps {
  id: string;
  children: React.ReactNode;
  isSelected: boolean;
  isDisabled?: boolean;
  onClick: (id: string) => void;
}

const TagContainer = styled.div<{ isSelected: boolean; isDisabled: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  height: 36px;
  gap: 6px;
  padding: 0 12px;
  border-radius: 32px;
  background-color: ${({ theme, isSelected }) =>
    isSelected ? theme.colors.primary : theme.colors.translucent_bg};
  cursor: ${({ isDisabled }) => (isDisabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ isDisabled }) => (isDisabled ? 0.5 : 1)};
  transition: background-color 0.2s ease-in-out, color 0.2s ease-in-out;

  ${({ isDisabled, theme }) =>
    !isDisabled &&
    css`
      &:hover {
        background-color: ${theme.colors.primary};
      }
    `}
`;

const Tag: React.FC<TagProps> = ({
  id,
  isSelected,
  isDisabled = false,
  onClick,
  children,
}) => {
  const handleClick = () => {
    if (!isDisabled) {
      onClick(id);
    }
  };

  return (
    <TagContainer
      isSelected={isSelected}
      isDisabled={isDisabled}
      onClick={handleClick}
      role="button"
      aria-disabled={isDisabled}
      aria-pressed={isSelected}
    >
      {children}
    </TagContainer>
  );
};

export { Tag };
