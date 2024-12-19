import React, { type ChangeEventHandler, type KeyboardEventHandler, useRef } from 'react';
import styled, { css } from 'styled-components';
import { FieldError, FieldLabel } from '@/reuseable-components';

interface TextAreaProps extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
  errorMessage?: string;
  title?: string;
  tooltip?: string;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  position: relative;
  width: 100%;
`;

const InputWrapper = styled.div<{ $disabled?: boolean; $hasError?: boolean; $isActive?: boolean }>`
  width: 100%;

  display: flex;
  align-items: center;
  gap: 12px;

  transition: border-color 0.3s;
  border-radius: 24px;
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

const StyledTextArea = styled.textarea`
  flex: 1;
  border: none;
  outline: none;
  background: none;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  padding: 12px 20px 0;
  font-family: ${({ theme }) => theme.font_family.primary};
  font-weight: 300;
  line-height: 22px;
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
`;

export const TextArea: React.FC<TextAreaProps> = ({ errorMessage, title, tooltip, required, onChange, name, ...props }) => {
  const ref = useRef<HTMLTextAreaElement>(null);

  const resize = (focused: boolean) => {
    // this is to auto-resize the textarea according to the number of rows typed
    if (ref.current) {
      ref.current.style.height = 'auto';
      if (focused) ref.current.style.height = `${ref.current.scrollHeight}px`;
    }
  };

  const handleChange: ChangeEventHandler<HTMLTextAreaElement> = (e) => {
    e.stopPropagation();
    resize(true);
    onChange?.(e);
  };

  const handleKeyDown: KeyboardEventHandler<HTMLTextAreaElement> = (e) => {
    e.stopPropagation();
  };

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      <InputWrapper $disabled={props.disabled} $hasError={!!errorMessage} $isActive={!!props.autoFocus}>
        <StyledTextArea ref={ref} data-id={name} name={name} onFocus={() => resize(true)} onBlur={() => resize(false)} onChange={handleChange} onKeyDown={handleKeyDown} {...props} />
      </InputWrapper>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </Container>
  );
};
