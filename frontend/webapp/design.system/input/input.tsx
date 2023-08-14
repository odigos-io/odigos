import React from "react";
import { Input } from "@keyval-dev/design-system";
interface InputProps {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  error?: string;
  style?: React.CSSProperties;
}

export function KeyvalInput(props: InputProps): JSX.Element {
  return <Input {...props} />;
}
