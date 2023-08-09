import React from "react";
import { SETUP } from "@/utils/constants";
import { Switch } from "@keyval-org/design-system";

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
    <Switch
      toggle={toggle}
      handleToggleChange={handleToggleChange}
      style={style}
      label={label}
    />
  );
}
