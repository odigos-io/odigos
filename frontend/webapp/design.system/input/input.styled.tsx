import { styled } from "styled-components";

interface ActiveProps {
  active?: any;
  hasError: boolean;
}

export const StyledInputContainer = styled.div<ActiveProps>`
  position: relative;
  display: flex;
  width: 100%;
  padding-left: 13px;
  height: 100%;
  align-items: center;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  gap: 10px;
  border-radius: 8px;
  border: ${({ theme, hasError, active }) =>
    `1px solid ${
      hasError
        ? theme.colors.error
        : active
        ? theme.text.grey
        : theme.colors.blue_grey
    }`};
  background: ${({ theme }) => theme.colors.light_dark};

  &:hover {
    border: ${({ theme }) => `solid 1px ${theme.text.grey}`};
  }
`;

export const StyledActionInputContainer = styled.div`
  position: relative;
  display: flex;
  width: 100%;
  padding: 0px 12px;
  height: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  border-radius: 4px;
  border: ${({ theme }) => `1px solid ${theme.colors.secondary}`};
`;

export const StyledInput = styled.input`
  background: transparent;
  border: none;
  outline: none;
  width: 100%;
  color: ${({ theme }) => theme.text.white};
`;

export const StyledActionInput = styled(StyledInput)`
  color: var(--dark-mode-white, #fff);
  font-family: Inter, sans-serif;
  font-size: 24px;
`;

export const LabelWrapper = styled.div`
  margin-bottom: 8px;
`;

export const ErrorWrapper = styled.div`
  margin-top: 4px;
`;

export const DisplayIconsWrapper = styled.div`
  position: absolute;
  right: 10px;
  cursor: pointer;
`;
