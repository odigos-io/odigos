import { styled } from "styled-components";

interface CheckboxWrapperProps {
  disabled?: boolean;
}

export const CheckboxWrapper = styled.div<CheckboxWrapperProps>`
  display: flex;
  gap: 8px;
  align-items: center;
  cursor: ${({ disabled }) => (disabled ? "not-allowed" : "pointer")};
  pointer-events: ${({ disabled }) => (disabled ? "none" : "auto")};
  opacity: ${({ disabled }) => (disabled ? "0.5" : "1")};
`;

export const Checkbox = styled.span`
  width: 16px;
  height: 16px;
  border: ${({ theme }) => `solid 1px ${theme.colors.light_grey}`};
  border-radius: 4px;
`;
