import React, { ChangeEvent } from "react";
import { StyledActionInputContainer, StyledActionInput } from "./input.styled";
import { KeyvalButton } from "../button/button";
import { KeyvalText } from "../text/text";
import theme from "@/styles/palette";
import { ACTION } from "@/utils/constants";

interface InputProps {
  value: string;
  onAction: () => void;
  onChange: (value: string) => void;
  type?: string;
  style?: React.CSSProperties;
}

export function KeyvalActionInput({
  value,
  onChange,
  style = {},
  onAction,
}: InputProps): JSX.Element {
  function handleChange(event: ChangeEvent<HTMLInputElement>): void {
    onChange(event.target.value);
  }

  return (
    <>
      <StyledActionInputContainer style={{ ...style }}>
        <StyledActionInput
          value={value}
          onChange={handleChange}
          autoComplete="off"
        />

        <KeyvalButton onClick={onAction}>
          <KeyvalText size={14} weight={500} color={theme.text.dark_button}>
            {ACTION.SAVE}
          </KeyvalText>
        </KeyvalButton>
      </StyledActionInputContainer>
    </>
  );
}
