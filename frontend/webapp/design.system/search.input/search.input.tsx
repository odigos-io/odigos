import React from "react";
import { SearchInput } from "@odigos-io/design-system";
interface KeyvalSearchInputProps {
  placeholder?: string;
  value?: string;
  onChange?: (e: any) => void;
  loading?: boolean;
  containerStyle?: any;
  inputStyle?: any;
  showClear?: boolean;
}

export function KeyvalSearchInput(props: KeyvalSearchInputProps) {
  return <SearchInput {...props} />;
}
