import React from "react";
import { KeyvalText } from "../text/text";
import styled from "styled-components";
import Checked from "@/assets/icons/checkbox-rect.svg";
interface KeyvalCheckboxProps {
  value: boolean;
  onChange: () => void;
  label?: string;
}

export const CheckboxWrapper = styled.div`
  display: flex;
  gap: 8px;
  align-items: center;
  cursor: pointer;
`;

export const Checkbox = styled.span`
  width: 16px;
  height: 16px;
  border: solid 1px #ccd0d2;
  border-radius: 4px;
`;

export function KeyvalCheckbox({
  onChange,
  value,
  label = "Select All",
}: KeyvalCheckboxProps) {
  return (
    <CheckboxWrapper onClick={onChange}>
      {value ? <Checked /> : <Checkbox />}
      <KeyvalText size={14}>{label}</KeyvalText>
    </CheckboxWrapper>
  );
}
