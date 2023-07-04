import React, { ButtonHTMLAttributes, FC } from "react";
import { ButtonWrapper } from "./button.styled";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  // Additional custom props if needed
  variant?: "primary" | "secondary";
}

export const KeyvalButton: FC<ButtonProps> = ({
  variant = "primary",
  children,
  style,
}) => {
  return (
    <ButtonWrapper style={{ ...style }} variant={variant}>
      {children}
    </ButtonWrapper>
  );
};
