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
  width: 100%;
  height: 100%;
  cursor: ${({ disabled }) =>
    disabled ? "not-allowed !important" : "pointer !important"};
  background: ${({ theme, disabled }) =>
    disabled ? theme.colors.blue_grey : theme.colors.secondary};
  justify-content: center;
  align-items: center;
`;
