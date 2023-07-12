import React from "react";
import { KeyvalText } from "../text/text";
import { CheckboxWrapper, Checkbox } from "./checkbox.styled";
import Checked from "@/assets/icons/checkbox-rect.svg";

interface KeyvalCheckboxProps {
  value: boolean;
  onChange: () => void;
  label?: string;
  disabled?: boolean;
}

export function KeyvalCheckbox({
  onChange,
  value,
  label = "",
  disabled = false,
}: KeyvalCheckboxProps) {
  return (
    <CheckboxWrapper disabled={disabled || undefined} onClick={onChange}>
      {value ? <Checked /> : <Checkbox />}
      <KeyvalText size={14}>{label}</KeyvalText>
    </CheckboxWrapper>
  );
}
