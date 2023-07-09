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
  placeholder = "Search - default",
  value = "",
  onChange = () => {},
  loading = false,
  containerStyle = {},
  inputStyle = {},
  showClear = true,
}: KeyvalSearchInputProps) {
  return (
    <SearchInputWrapper active={!!value} style={{ ...containerStyle }}>
      <Glass />
      <StyledSearchInput
        style={{ ...inputStyle }}
        value={value}
        active={!!value}
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
      {showClear && (
        <X
          onClick={
            value
              ? () =>
                  onChange({
                    target: {
                      value: "",
                    },
                  })
              : null
          }
          style={{ cursor: "pointer" }}
        />
      )}
    </SearchInputWrapper>
  );
}
