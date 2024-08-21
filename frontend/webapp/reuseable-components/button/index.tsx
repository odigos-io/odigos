import React, { ButtonHTMLAttributes } from 'react';
import styled, { css } from 'styled-components';

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'tertiary';
  isDisabled?: boolean;
}

const variantStyles = {
  primary: css`
    border-radius: 32px;
    border: 1px solid rgba(249, 249, 249, 0.24);
    background: rgba(249, 249, 249, 0.8);
    height: 36px;
    padding: 8px 14px 8px 16px;

    &:hover {
      background: rgba(249, 249, 249, 0.6);
    }
    &:active {
      background: rgba(249, 249, 249, 0.5);
    }
  `,
  secondary: css`
    background: #151515;
    border: 1px solid rgba(249, 249, 249, 0.24);
    border-radius: 32px;
    &:hover {
      border: 1px solid rgba(249, 249, 249, 0.32);
      background: rgba(249, 249, 249, 0.04);
    }
    &:active {
      background: #1515158d;
    }
  `,
  tertiary: css`
    background-color: transparent;
    border-radius: 32px;
    &:hover {
      background: #151515;
    }
    &:active {
    }
  `,
};

const StyledButton = styled.button<ButtonProps>`
  padding: 10px 20px;
  border: none;
  border-radius: 5px;
  cursor: pointer;
  transition: background-color 0.3s ease;

  ${({ variant }) => variant && variantStyles[variant]}
  ${({ isDisabled }) =>
    isDisabled &&
    css`
      opacity: 0.5;
      cursor: not-allowed;
      &:hover,
      &:active {
        background-color: #eaeaea;
      }
    `}
`;

export const Button: React.FC<ButtonProps> = ({
  children,
  variant = 'primary',
  isDisabled = false,
  ...props
}) => {
  return (
    <StyledButton variant={variant} isDisabled={isDisabled} {...props}>
      {children}
    </StyledButton>
  );
};
