import React, { FC, ReactNode } from 'react';
import { Button } from '@keyval-dev/design-system';
interface ButtonProps {
  variant?: string;
  children: JSX.Element | JSX.Element[];
  onClick?: () => void;
  style?: object;
  disabled?: boolean;
  type?: 'button' | 'submit' | 'reset' | undefined;
}
export const KeyvalButton: FC<ButtonProps> = (props) => {
  return <Button {...props}>{props.children}</Button>;
};
