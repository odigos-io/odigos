import React from "react";
import { Switch } from "@keyval-org/design-system";

interface KeyvalSwitchProps {
  toggle: boolean;
  handleToggleChange: () => void;
  style?: object;
  label?: string;
}

export function KeyvalSwitch(props: KeyvalSwitchProps) {
  return <Switch {...props} />;
}
