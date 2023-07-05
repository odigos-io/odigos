import React from "react";
import { SearchInputWrapper, StyledSearchInput } from "./search.input.styled";
import Glass from "@/assets/icons/glass.svg";
import X from "@/assets/icons/X.svg";

interface KeyvalSearchInputProps {
  placeholder?: string;
  value?: string;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
}

export function KeyvalSearchInput({
  placeholder = "Search - default",
  value = "sdfd",
  onChange = () => {},
}: KeyvalSearchInputProps) {
  return (
    <SearchInputWrapper active={!!value}>
      <Glass />
      <StyledSearchInput
        value={value}
        active={!!value}
        placeholder={placeholder}
      />
      <X onClick={() => onChange(null)} style={{ cursor: "pointer" }} />
    </SearchInputWrapper>
  );
}
