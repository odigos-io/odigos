import styled from "styled-components";

interface ButtonProps {
  variant?: string;
}

export const ButtonWrapper = styled.button<ButtonProps>`
  display: flex;
  padding: 8px 16px;
  align-items: center;
  border-radius: 8px;
  border: none;
  background: ${({ theme }) => theme.colors.secondary};
`;
