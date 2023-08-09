import React from "react";
import { ActionInput } from "@keyval-org/design-system";

interface InputProps {
  value: string;
  onAction: () => void;
  onChange: (value: string) => void;
  type?: string;
  style?: React.CSSProperties;
}

export function KeyvalActionInput(props: InputProps): JSX.Element {
  return <ActionInput {...props} />;
}
