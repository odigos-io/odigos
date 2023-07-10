import React from "react";
import {
  LoaderWrapper,
  SearchInputWrapper,
  StyledSearchInput,
} from "./search.input.styled";
import Glass from "@/assets/icons/glass.svg";
import X from "@/assets/icons/X.svg";
import { KeyvalLottie } from "../lottie/lottie";
import loader from "@/assets/lotties/loader.json";
import { SETUP } from "@/utils/constants";
interface KeyvalSearchInputProps {
  placeholder?: string;
  value?: string;
  onChange?: (e: any) => void;
  loading?: boolean;
  containerStyle?: any;
  inputStyle?: any;
  showClear?: boolean;
}

export function KeyvalSearchInput({
  placeholder = SETUP.MENU.SEARCH_PLACEHOLDER,
  value = "",
  onChange = () => {},
  loading = false,
  containerStyle = {},
  inputStyle = {},
  showClear = true,
}: KeyvalSearchInputProps) {
  const clear = value
    ? () =>
        onChange({
          target: {
            value: "",
          },
        })
    : null;

  return (
    <SearchInputWrapper
      active={!!value || undefined}
      style={{ ...containerStyle }}
    >
      <Glass />
      <StyledSearchInput
        style={{ ...inputStyle }}
        value={value}
        active={!!value || undefined}
        placeholder={placeholder}
        onChange={onChange}
      />
      {loading && (
        <LoaderWrapper>
          <KeyvalLottie
            animationData={loader}
            autoplay
            loop
            width={26}
            height={26}
          />
        </LoaderWrapper>
      )}
      {showClear && <X onClick={clear} style={{ cursor: "pointer" }} />}
    </SearchInputWrapper>
  );
}
