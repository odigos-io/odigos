import React, { useState, forwardRef } from 'react';
import Image from 'next/image';
import styled, { css } from 'styled-components';
import { FieldError, FieldLabel } from '@/reuseable-components';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  title?: string;
  icon?: string;
  tooltip?: string;
  initialValue?: string;
  buttonLabel?: string;
  onButtonClick?: () => void;
  required?: boolean;
  hasError?: boolean; // this is to apply error styles without using an error message
  errorMessage?: string;
}

// Styled components remain the same as before
const Container = styled.div`
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
`;

const InputWrapper = styled.div<{ $disabled?: InputProps['disabled']; $hasError?: boolean; $isActive?: boolean }>`
  width: 100%;
  display: flex;
  align-items: center;
  height: 36px;
  gap: 12px;
  transition: border-color 0.3s;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);
  ${({ $disabled }) =>
    $disabled &&
    css`
      background-color: #555;
      cursor: not-allowed;
      opacity: 0.6;
    `}
  ${({ $hasError }) =>
    $hasError &&
    css`
      border-color: ${({ theme }) => theme.text.error};
    `}
  ${({ $isActive }) =>
    $isActive &&
    css`
      border-color: ${({ theme }) => theme.colors.secondary};
    `}
  &:hover {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
  &:focus-within {
    border-color: ${({ theme }) => theme.colors.secondary};
  }
`;

const StyledInput = styled.input<{ $hasIcon: boolean }>`
  padding-left: ${({ $hasIcon }) => ($hasIcon ? '0' : '16px')};
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
  &::-webkit-inner-spin-button,
  &::-webkit-outer-spin-button {
    -webkit-appearance: none;
    margin: 0;
  }
`;

const IconWrapper = styled.div`
  display: flex;
  align-items: center;
  margin-left: 12px;
`;

const IconWrapperClickable = styled(IconWrapper)`
  cursor: pointer;
`;

const Button = styled.button`
  background-color: ${({ theme }) => theme.colors.primary};
  border: none;
  color: #fff;
  padding: 8px 16px;
  border-radius: 20px;
  cursor: pointer;
  margin-left: 8px;
  &:hover {
    background-color: ${({ theme }) => theme.colors.secondary};
  }
  &:disabled {
    background-color: #555;
    cursor: not-allowed;
  }
`;

// Wrap Input with forwardRef to handle the ref prop
const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ icon, buttonLabel, onButtonClick, hasError, errorMessage, title, tooltip, required, initialValue, onChange, type = 'text', name, ...props }, ref) => {
    const isSecret = type === 'password';
    const [revealSecret, setRevealSecret] = useState(false);
    const [value, setValue] = useState<string>(initialValue || '');

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      setValue(e.target.value);
      if (onChange) {
        onChange(e);
      }
    };

    return (
      <Container>
        <FieldLabel title={title} required={required} tooltip={tooltip} />

        <InputWrapper $disabled={props.disabled} $hasError={hasError || !!errorMessage} $isActive={!!props.autoFocus}>
          {isSecret ? (
            <IconWrapperClickable onClick={() => setRevealSecret((prev) => !prev)}>
              <Image src={revealSecret ? '/icons/common/eye-closed.svg' : '/icons/common/eye-open.svg'} alt='' width={14} height={14} />
            </IconWrapperClickable>
          ) : icon ? (
            <IconWrapper>
              <Image src={icon} alt='' width={14} height={14} />
            </IconWrapper>
          ) : null}

          <StyledInput
            ref={ref} // Pass ref to the StyledInput
            data-id={name}
            name={name}
            $hasIcon={!!icon || isSecret}
            value={value}
            onChange={handleInputChange}
            type={revealSecret ? 'text' : type}
            {...props}
          />

          {buttonLabel && onButtonClick && (
            <Button onClick={onButtonClick} disabled={props.disabled}>
              {buttonLabel}
            </Button>
          )}
        </InputWrapper>

        {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
      </Container>
    );
  },
);

Input.displayName = 'Input'; // Set a display name for easier debugging
export { Input };
