import React, { FC, ChangeEvent } from "react";

import { RadioButton } from "@odigos-io/design-system";
interface RadioButtonProps {
  label?: string;
  value?: string;
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
}

export const KeyvalRadioButton: FC<RadioButtonProps> = (props) => {
  return <RadioButton {...props} />;
};
