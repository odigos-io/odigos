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
}

export function KeyvalInput({
  label,
  value,
  onChange,
  type = "text",
  error,
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
      <StyledInputContainer hasError={!!error}>
        <StyledInput type={type} value={value} onChange={handleChange} />
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
