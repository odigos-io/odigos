import React, { ChangeEvent } from "react";
import {
  StyledInputContainer,
  StyledInput,
  ErrorWrapper,
  LabelWrapper,
} from "./input.styled";
import { KeyvalText } from "../text/text";

interface InputProps {
  label?: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  error?: string;
  style?: React.CSSProperties;
}

export function KeyvalInput({
  label,
  value,
  onChange,
  type = "text",
  error,
  style = {},
}: InputProps): JSX.Element {
  function handleChange(event: ChangeEvent<HTMLInputElement>): void {
    onChange(event.target.value);
  }

  return (
    <>
      {label && (
        <LabelWrapper>
          <KeyvalText size={14} weight={600}>
            {label}
          </KeyvalText>
        </LabelWrapper>
      )}
      <StyledInputContainer hasError={!!error} style={{ ...style }}>
        <StyledInput
          type={type}
          value={value}
          onChange={handleChange}
          autoComplete="off"
        />
      </StyledInputContainer>
      {error && (
        <ErrorWrapper>
          <KeyvalText size={14} color={"#FD3F3F"}>
            {"error message text"}
          </KeyvalText>
        </ErrorWrapper>
      )}
    </>
  );
}
