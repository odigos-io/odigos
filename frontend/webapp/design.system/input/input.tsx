import React from 'react';
import { Input } from '@keyval-dev/design-system';
interface InputProps {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  error?: string;
  style?: React.CSSProperties;
  required?: boolean;
  placeholder?: string;
  tooltip?: string;
  min?: number;
  max?: number;
}

export function KeyvalInput(props: InputProps): JSX.Element {
  return <Input {...props} />;
}
