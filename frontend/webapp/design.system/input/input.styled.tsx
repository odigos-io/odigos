import { styled } from "styled-components";

export const StyledInputContainer = styled.div<{ hasError: boolean }>`
  display: flex;
  width: 360px;
  padding: 11px 13px;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  gap: 10px;
  border-radius: 8px;
  border: ${({ theme, hasError }) =>
    `1px solid ${hasError ? "#FD3F3F" : theme.text.grey}`};
  background: ${({ theme }) => theme.colors.light_dark};
`;

export const StyledInput = styled.input`
  background: transparent;
  border: none;
  outline: none;
  width: 100%;
  color: ${({ theme }) => theme.text.white};
`;

export const LabelWrapper = styled.div`
  margin-bottom: 11px;
`;

export const ErrorWrapper = styled.div`
  margin-top: 4px;
`;
