import React, { useState, forwardRef, type ChangeEvent, type KeyboardEventHandler, type InputHTMLAttributes } from 'react';
import styled, { css, useTheme } from 'styled-components';
import { EyeClosedIcon, EyeOpenIcon, SVG } from '@/assets';
import { FieldError, FieldLabel } from '@/reuseable-components';

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  title?: string;
  icon?: SVG;
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
  border: 1px solid ${({ theme }) => theme.colors.border};
  ${({ $disabled }) =>
    $disabled &&
    css`
      background-color: ${({ theme }) => theme.colors.border};
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
  color: ${({ theme }) => theme.text.secondary};
  font-size: 14px;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-weight: 300;
  &::placeholder {
    color: ${({ theme }) => theme.text.secondary};
    font-family: ${({ theme }) => theme.font_family.primary};
    opacity: 0.4;
    font-size: 14px;
    font-weight: 300;
    line-height: 22px; /* 157.143% */
  }
  &:disabled {
    background-color: ${({ theme }) => theme.colors.border};
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
  color: ${({ theme }) => theme.text.white};
  padding: 8px 16px;
  border-radius: 20px;
  cursor: pointer;
  margin-left: 8px;
  &:hover {
    background-color: ${({ theme }) => theme.colors.secondary};
  }
  &:disabled {
    background-color: ${({ theme }) => theme.colors.border};
    cursor: not-allowed;
  }
`;

// Wrap Input with forwardRef to handle the ref prop
const Input = forwardRef<HTMLInputElement, InputProps>(({ icon: Icon, buttonLabel, onButtonClick, hasError, errorMessage, title, tooltip, required, onChange, type = 'text', name, ...props }, ref) => {
  const theme = useTheme();

  const isSecret = type === 'password';
  const [revealSecret, setRevealSecret] = useState(false);

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation();

    const v = e.target.value;
    const actualValue = type === 'number' ? v.replace(/[^\d]/g, '') : v;
    e.target.value = actualValue;

    onChange?.(e);
  };

  const handleKeyDown: KeyboardEventHandler<HTMLInputElement> = (e) => {
    if (!['Enter'].includes(e.key)) e.stopPropagation();
  };

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      <InputWrapper $disabled={props.disabled} $hasError={hasError || !!errorMessage} $isActive={!!props.autoFocus}>
        {isSecret ? (
          <IconWrapperClickable onClick={() => setRevealSecret((prev) => !prev)}>
            {revealSecret ? <EyeClosedIcon size={14} fill={theme.text.grey} /> : <EyeOpenIcon size={14} fill={theme.text.grey} />}
          </IconWrapperClickable>
        ) : Icon ? (
          <IconWrapper>
            <Icon size={14} fill={theme.text.grey} />
          </IconWrapper>
        ) : null}

        <StyledInput ref={ref} data-id={name} type={revealSecret ? 'text' : type} $hasIcon={!!Icon || isSecret} name={name} onChange={handleInputChange} onKeyDown={handleKeyDown} {...props} />

        {buttonLabel && onButtonClick && (
          <Button onClick={onButtonClick} disabled={props.disabled}>
            {buttonLabel}
          </Button>
        )}
      </InputWrapper>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </Container>
  );
});

Input.displayName = 'Input';
export { Input };
