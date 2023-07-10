import React, { ButtonHTMLAttributes, FC } from "react";
import { StyledButton, ButtonContainer } from "./button.styled";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  // Additional custom props if needed
  variant?: "primary" | "secondary";
}

export const KeyvalButton: FC<ButtonProps> = ({
  variant = "primary",
  children,
  style,
  onClick,
  disabled,
}) => {
  return (
    <ButtonContainer disabled={disabled}>
      <StyledButton disabled={disabled} onClick={onClick} style={{ ...style }}>
        {children}
      </StyledButton>
    </ButtonContainer>
  );
};
