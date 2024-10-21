import React, { ButtonHTMLAttributes } from 'react';
import styled, { css } from 'styled-components';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'tertiary' | 'danger';
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
    color: ${({ theme }) => theme.colors.secondary};
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
  danger: css`
    border-color: transparent;
    background: ${({ theme }) => theme.colors.danger};
    &:hover {
      background: ${({ theme }) => theme.colors.danger};
      opacity: 0.9;
    }
    &:active {
      background: ${({ theme }) => theme.colors.danger};
    }
    &:focus {
      background: ${({ theme }) => theme.colors.danger};
    }
  `,
};

const StyledButton = styled.button<ButtonProps>`
  height: 36px;
  border-radius: 32px;
  cursor: pointer;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 0 12px;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-transform: uppercase;
  text-decoration: underline;
  font-weight: 600;
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
  variant?: 'primary' | 'secondary' | 'tertiary' | 'danger';
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
