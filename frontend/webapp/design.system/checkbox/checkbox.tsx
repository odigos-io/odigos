import React from "react";
import { Checkbox } from "@keyval-dev/design-system";

interface KeyvalCheckboxProps {
  value: boolean;
  onChange: () => void;
  label?: string;
  disabled?: boolean;
}

export function KeyvalCheckbox(props: KeyvalCheckboxProps) {
  return <Checkbox {...props} />;
}
