import React, { ButtonHTMLAttributes, forwardRef } from 'react';
import { hexPercentValues } from '@/styles';
import styled, { css } from 'styled-components';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'tertiary' | 'danger' | 'warning';
}

const variantStyles = {
  primary: css`
    border: 1px solid ${({ theme }) => theme.text.secondary + hexPercentValues['024']};
    background: ${({ theme }) => theme.colors.secondary};
    &:hover {
      background: #e0e0e0;
    }
    &:active {
      background: ${({ theme }) => theme.text.grey};
    }
    &:focus {
      background: ${({ theme }) => theme.colors.secondary};
    }
  `,
  secondary: css`
    background: ${({ theme }) => theme.text.secondary + hexPercentValues['000']};
    border: 1px solid ${({ theme }) => theme.colors.border};
    color: ${({ theme }) => theme.colors.secondary};
    &:hover {
      border: 1px solid ${({ theme }) => theme.colors.white_opacity['30']};
      background: ${({ theme }) => theme.colors.white_opacity['004']};
    }
    &:active {
      background: ${({ theme }) => theme.colors.white_opacity['008']};
      border: 1px solid ${({ theme }) => theme.text.dark_grey};
    }
    &:focus {
      background: ${({ theme }) => theme.text.secondary + hexPercentValues['000']};
    }
  `,
  tertiary: css`
    border-color: transparent;
    background: transparent;
    &:hover {
      background: ${({ theme }) => theme.colors.white_opacity['004']};
    }
    &:active {
      background: ${({ theme }) => theme.colors.white_opacity['008']};
    }
    &:focus {
      background: ${({ theme }) => theme.text.secondary + hexPercentValues['000']};
    }
  `,
  danger: css`
    border-color: transparent;
    background: ${({ theme }) => theme.text.error};
    &:hover {
      background: ${({ theme }) => theme.text.error};
      opacity: 0.9;
    }
    &:active {
      background: ${({ theme }) => theme.text.error};
    }
    &:focus {
      background: ${({ theme }) => theme.text.error};
    }
  `,
  warning: css`
    border-color: transparent;
    background: ${({ theme }) => theme.text.warning};
    &:hover {
      background: ${({ theme }) => theme.text.warning};
      opacity: 0.9;
    }
    &:active {
      background: ${({ theme }) => theme.text.warning};
    }
    &:focus {
      background: ${({ theme }) => theme.text.warning};
    }
  `,
};

const StyledButton = styled.button<{ $variant: ButtonProps['variant'] }>`
  height: 36px;
  border-radius: 32px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 0 12px;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-transform: uppercase;
  text-decoration: underline;
  font-weight: 600;
  outline: none;
  ${({ $variant }) => $variant && variantStyles[$variant]}
  ${({ disabled, $variant }) =>
    disabled &&
    css`
      opacity: 0.5;
      cursor: not-allowed;

      ${$variant === 'primary'
        ? css`
            color: ${({ theme }) => theme.colors.secondary};
            background: ${({ theme }) => theme.text.secondary + hexPercentValues['010']};
            &:hover {
              background: ${({ theme }) => theme.text.secondary + hexPercentValues['015']};
            }
          `
        : ''}
    `}
`;

const ButtonContainer = styled.div<{ $variant: ButtonProps['variant'] }>`
  height: fit-content;
  border: 2px solid transparent;
  padding: 2px;
  border-radius: 32px;
  background-color: transparent;
  transition: border-color 0.3s ease;
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(({ children, variant = 'primary', ...props }, ref) => {
  return (
    <ButtonContainer $variant={variant}>
      <StyledButton ref={ref} $variant={variant} {...props}>
        {children}
      </StyledButton>
    </ButtonContainer>
  );
});
