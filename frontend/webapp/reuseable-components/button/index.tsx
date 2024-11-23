import React, { ButtonHTMLAttributes, forwardRef, LegacyRef } from 'react';
import styled, { css } from 'styled-components';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'tertiary' | 'danger' | 'warning';
  isDisabled?: boolean; // ??? do we need this, i think we can use "disabled" default HTML Button attribute
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
  ${({ disabled }) =>
    disabled &&
    css`
      opacity: 0.5;
      cursor: not-allowed;
      &:hover {
        background-color: #eaeaea;
      }
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

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(({ children, variant = 'primary', isDisabled = false, ...props }, ref) => {
  return (
    <ButtonContainer $variant={variant}>
      <StyledButton ref={ref} $variant={variant} disabled={isDisabled || props.disabled} {...props}>
        {children}
      </StyledButton>
    </ButtonContainer>
  );
});
