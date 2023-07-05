import React from "react";
import { KeyvalText } from "../text/text";
import {
  SwitchButtonWrapper,
  SwitchInputWrapper,
  SwitchToggleWrapper,
} from "./switch.styled";

interface KeyvalSwitchProps {
  toggle: boolean;
  handleToggleChange: () => void;
  style?: object;
  label?: string;
}

export function KeyvalSwitch({
  toggle,
  handleToggleChange,
  style,
  label = "Select All",
}: KeyvalSwitchProps) {
  return (
    <SwitchInputWrapper>
      <SwitchToggleWrapper active={toggle} onClick={handleToggleChange}>
        <SwitchButtonWrapper disabled={toggle} />
      </SwitchToggleWrapper>
      {label && <KeyvalText size={14}>{label}</KeyvalText>}
    </SwitchInputWrapper>
  );
}
