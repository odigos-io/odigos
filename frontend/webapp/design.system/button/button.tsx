import React, { FC, ReactNode } from "react";
import { Button } from "@keyval-dev/design-system";
interface ButtonProps {
  variant?: "primary" | "secondary";
  children: ReactNode;
  onClick?: () => void;
  style?: object;
  disabled?: boolean;
}

export const KeyvalButton: FC<ButtonProps> = (props) => {
  return <Button {...props}>{props.children}</Button>;
};
