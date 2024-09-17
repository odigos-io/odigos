import React, { ButtonHTMLAttributes } from 'react';
import styled, { css } from 'styled-components';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'tertiary';
  isDisabled?: boolean;
}

const variantStyles = {
  primary: css`
    border: 1px solid rgba(249, 249, 249, 0.24);
    background: ${({ theme }) => theme.colors.secondary};
    &:hover {
      background: rgba(224, 224, 224, 1);
    }
    &:active {
      background: rgba(184, 184, 184, 1);
    }
    &:focus {
      background: ${({ theme }) => theme.colors.secondary};
    }
  `,
  secondary: css`
    background: rgba(249, 249, 249, 0);
    border: 1px solid rgba(82, 82, 82, 1);

    &:hover {
      border: 1px solid rgba(249, 249, 249, 0.32);
      background: rgba(249, 249, 249, 0.04);
    }
    &:active {
      background: rgba(249, 249, 249, 0.08);
      border: 1px solid rgba(143, 143, 143, 1);
    }
    &:focus {
      background: rgba(249, 249, 249, 0);
    }
  `,
  tertiary: css`
    border-color: transparent;
    background: transparent;
    &:hover {
      background: rgba(249, 249, 249, 0.04);
    }
    &:active {
      background: rgba(249, 249, 249, 0.08);
    }
    &:focus {
      background: rgba(249, 249, 249, 0);
    }
  `,
};

const StyledButton = styled.button<ButtonProps>`
  height: 36px;
  border-radius: 32px;
  cursor: pointer;
  transition: background-color 0.3s ease;
  padding: 0 12px;
  ${({ variant }) => variant && variantStyles[variant]}
  ${({ isDisabled }) =>
    isDisabled &&
    css`
      opacity: 0.5;
      cursor: not-allowed;
      &:hover {
        background-color: #eaeaea;
      }
    `}
`;

const ButtonContainer = styled.div<{
  variant?: 'primary' | 'secondary' | 'tertiary';
}>`
  border: 2px solid transparent;
  padding: 2px;
  border-radius: 32px;
  background-color: transparent;
  transition: border-color 0.3s ease;
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  isDisabled = false,
  ...props
}) => {
  return (
    <ButtonContainer variant={variant}>
      <StyledButton
        variant={variant}
        disabled={isDisabled}
        isDisabled={isDisabled}
        {...props}
      >
        {children}
      </StyledButton>
    </ButtonContainer>
  );
};
