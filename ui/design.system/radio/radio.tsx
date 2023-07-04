import React, { useState, FC, ChangeEvent } from "react";
import { RadioButtonContainer, RadioButtonBorder } from "./radio.styled";
import { KeyvalText } from "design.system/text/text";
import Checked from "assets/icons/checked-radio.svg";
interface RadioButtonProps {
  label?: string;
  value?: string;
  checked?: boolean;
  onChange?: (event: ChangeEvent<HTMLInputElement>) => void;
}

export const KeyvalRadioButton: FC<RadioButtonProps> = ({
  label = "",
  value,
  //   checked = true,
  onChange,
}) => {
  const [checked, setChecked] = useState(false);

  function handleChange() {
    setChecked(!checked);
  }

  return (
    <RadioButtonContainer>
      <div onClick={handleChange}>
        {checked ? <Checked width={25} height={25} /> : <RadioButtonBorder />}
      </div>
      <KeyvalText>{label}</KeyvalText>
    </RadioButtonContainer>
  );
};
