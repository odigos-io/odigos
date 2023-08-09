import React from "react";
import { Checkbox } from "@keyval-org/design-system";

interface KeyvalCheckboxProps {
  value: boolean;
  onChange: () => void;
  label?: string;
  disabled?: boolean;
}

export function KeyvalCheckbox(props: KeyvalCheckboxProps) {
  return <Checkbox {...props} />;
}
