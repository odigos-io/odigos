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
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  loading?: boolean;
}

export function KeyvalSearchInput({
  placeholder = "Search - default",
  value = "test",
  onChange = () => {},
  loading = true,
}: KeyvalSearchInputProps) {
  return (
    <SearchInputWrapper active={!!value}>
      <Glass />
      <StyledSearchInput
        value={value}
        active={!!value}
        placeholder={placeholder}
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
      <X onClick={() => onChange(null)} style={{ cursor: "pointer" }} />
    </SearchInputWrapper>
  );
}
