import React, { ChangeEvent, useState } from "react";
import {
  StyledInputContainer,
  StyledInput,
  ErrorWrapper,
  LabelWrapper,
  DisplayIconsWrapper,
} from "./input.styled";
import { KeyvalText } from "../text/text";
import EyeOpenIcon from "@/assets/icons/design.system/eye-open.svg";
import EyeCloseIcon from "@/assets/icons/design.system/eye-close.svg";
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
  const [showPassword, setShowPassword] = useState<boolean>(false);

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
      <StyledInputContainer
        active={!!value || undefined}
        hasError={!!error}
        style={{ ...style }}
      >
        <StyledInput
          type={showPassword ? "text" : type}
          value={value}
          onChange={handleChange}
          autoComplete="off"
        />
        {type === "password" && (
          <DisplayIconsWrapper onClick={() => setShowPassword(!showPassword)}>
            {!showPassword ? (
              <EyeOpenIcon width={16} height={16} />
            ) : (
              <EyeCloseIcon width={16} height={16} />
            )}
          </DisplayIconsWrapper>
        )}
      </StyledInputContainer>
      {error && (
        <ErrorWrapper>
          <KeyvalText size={14} color={"#FD3F3F"}>
            {error}
          </KeyvalText>
        </ErrorWrapper>
      )}
    </>
  );
}
