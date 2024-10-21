import React from 'react';
import Image from 'next/image';
import { Text } from '../text';
import { Tooltip } from '../tooltip';
import styled, { css } from 'styled-components';

interface TextAreaProps
  extends React.TextareaHTMLAttributes<HTMLTextAreaElement> {
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

const InputWrapper = styled.div<{
  isDisabled?: boolean;
  hasError?: boolean;
  isActive?: boolean;
}>`
  width: 100%;

  display: flex;
  align-items: center;
  gap: 12px;

  transition: border-color 0.3s;
  border-radius: 24px;
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

const StyledTextArea = styled.textarea<{}>`
  flex: 1;
  border: none;
  outline: none;
  background: none;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  padding: 12px 20px;
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

const TextArea: React.FC<TextAreaProps> = ({
  errorMessage,
  title,
  tooltip,
  ...props
}) => {
  return (
    <Container>
      {title && (
        <HeaderWrapper>
          <Title>{title}</Title>
          <Tooltip text={tooltip || ''}>
            {tooltip && (
              <Image
                src="/icons/common/info.svg"
                alt=""
                width={16}
                height={16}
                style={{ marginBottom: 4 }}
              />
            )}
          </Tooltip>
        </HeaderWrapper>
      )}

      <InputWrapper
        isDisabled={props.disabled}
        hasError={!!errorMessage}
        isActive={!!props.autoFocus}
      >
        <StyledTextArea {...props} />
      </InputWrapper>
      {errorMessage && (
        <ErrorWrapper>
          <ErrorMessage>{errorMessage}</ErrorMessage>
        </ErrorWrapper>
      )}
    </Container>
  );
};

export { TextArea };
