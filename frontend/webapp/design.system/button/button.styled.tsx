import styled from "styled-components";

interface ButtonProps {
  variant?: string;
  disabled?: boolean;
}

export const ButtonContainer = styled.div<ButtonProps>`
  :hover {
    background: ${({ theme, disabled }) =>
      disabled ? theme.colors.blue_grey : theme.colors.torquiz_light};
  }
  p {
    cursor: ${({ disabled }) =>
      disabled ? "not-allowed !important" : "pointer !important"};
  }
`;

export const StyledButton = styled.button<ButtonProps>`
  display: flex;
  padding: 8px 16px;
  align-items: center;
  border-radius: 8px;
  border: none;
  cursor: ${({ disabled }) =>
    disabled ? "not-allowed !important" : "pointer !important"};
  background: ${({ theme, disabled }) =>
    disabled ? theme.colors.blue_grey : theme.colors.secondary};
`;
