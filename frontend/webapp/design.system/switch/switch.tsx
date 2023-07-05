import React from "react";
import "./switch.css";
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
}

export function KeyvalSwitch({
  toggle,
  handleToggleChange,
  style,
}: KeyvalSwitchProps) {
  return (
    <SwitchInputWrapper>
      <SwitchToggleWrapper active={toggle} onClick={handleToggleChange}>
        <SwitchButtonWrapper disabled={toggle} />
      </SwitchToggleWrapper>
      <KeyvalText size={14}>{"label"}</KeyvalText>
    </SwitchInputWrapper>
  );
}
