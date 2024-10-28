import Image from 'next/image';
import React, { useState, forwardRef } from 'react';
import { Text } from '../text';
import styled, { css } from 'styled-components';
import { Tooltip } from '../tooltip';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  icon?: string;
  buttonLabel?: string;
  onButtonClick?: () => void;
  errorMessage?: string;
  title?: string;
  tooltip?: string;
  required?: boolean;
  initialValue?: string;
  maxWidth?: string;
  paddingLeft?: string;
}

// Styled components remain the same as before
const Container = styled.div<{ maxWidth?: string }>`
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
  max-width: ${({ maxWidth }) => maxWidth || 'unset'};
`;

const InputWrapper = styled.div<{
  isDisabled?: boolean;
  hasError?: boolean;
  isActive?: boolean;
}>`
  width: 100%;
  display: flex;
  align-items: center;
  height: 36px;
  gap: 12px;
  transition: border-color 0.3s;
  border-radius: 32px;
  border: 1px solid rgba(249, 249, 249, 0.24);
  ${({ isDisabled }) =>
    isDisabled &&
    css`
      background-color: #555;
      cursor: not-allowed;
      opacity: 0.6;
    `}
  ${({ hasError }) =>
    hasError &&
    css`
      border-color: red;
    `}
  ${({ isActive }) =>
    isActive &&
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

const StyledInput = styled.input<{ hasIcon?: string; maxWidth?: string; paddingLeft?: string }>`
  max-width: ${({ maxWidth }) => maxWidth || 'unset'};
  padding-left: ${({ hasIcon, paddingLeft }) => (hasIcon ? '0' : paddingLeft || '16px')};
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

const ErrorWrapper = styled.div`
  position: relative;
`;

const ErrorMessage = styled(Text)`
  color: red;
  font-size: 12px;
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
`;

const Title = styled(Text)`
  font-size: 14px;
  opacity: 0.8;
  line-height: 22px;
  margin-bottom: 4px;
`;

const HeaderWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
`;

// Wrap Input with forwardRef to handle the ref prop
const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ icon, buttonLabel, onButtonClick, errorMessage, title, tooltip, required, initialValue, onChange, maxWidth, paddingLeft, ...props }, ref) => {
    const [value, setValue] = useState<string>(initialValue || '');

    const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      setValue(e.target.value);
      if (onChange) {
        onChange(e);
      }
    };

    return (
      <Container maxWidth={maxWidth}>
        {title && (
          <HeaderWrapper>
            <Title>{title}</Title>
            {!required && (
              <Text color='#7A7A7A' size={14} weight={300} opacity={0.8}>
                (optional)
              </Text>
            )}
            <Tooltip text={tooltip || ''}>
              {tooltip && <Image src='/icons/common/info.svg' alt='' width={16} height={16} style={{ marginBottom: 4 }} />}
            </Tooltip>
          </HeaderWrapper>
        )}

        <InputWrapper isDisabled={props.disabled} hasError={!!errorMessage} isActive={!!props.autoFocus}>
          {icon && (
            <IconWrapper>
              <Image src={icon} alt='' width={14} height={14} />
            </IconWrapper>
          )}
          <StyledInput
            ref={ref} // Pass ref to the StyledInput
            maxWidth={maxWidth}
            paddingLeft={paddingLeft}
            hasIcon={icon}
            value={value}
            onChange={handleInputChange}
            {...props}
          />
          {buttonLabel && onButtonClick && (
            <Button onClick={onButtonClick} disabled={props.disabled}>
              {buttonLabel}
            </Button>
          )}
        </InputWrapper>
        {errorMessage && (
          <ErrorWrapper>
            <ErrorMessage>{errorMessage}</ErrorMessage>
          </ErrorWrapper>
        )}
      </Container>
    );
  }
);

Input.displayName = 'Input'; // Set a display name for easier debugging
export { Input };
