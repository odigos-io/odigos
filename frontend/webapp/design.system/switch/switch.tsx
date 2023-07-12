import React from "react";
import { KeyvalText } from "../text/text";
import {
  SwitchButtonWrapper,
  SwitchInputWrapper,
  SwitchToggleWrapper,
} from "./switch.styled";
import { SETUP } from "@/utils/constants";

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
  label = SETUP.MENU.SELECT_ALL,
}: KeyvalSwitchProps) {
  return (
    <SwitchInputWrapper>
      <SwitchToggleWrapper
        active={toggle || undefined}
        onClick={handleToggleChange}
      >
        <SwitchButtonWrapper disabled={toggle || undefined} />
      </SwitchToggleWrapper>
      {label && <KeyvalText size={14}>{label}</KeyvalText>}
    </SwitchInputWrapper>
  );
}
