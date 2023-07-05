import React from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
} from "./sources.option.menu.styled";
import { KeyvalDropDown, KeyvalSearchInput, KeyvalText } from "@/design.system";

export function SourcesOptionMenu() {
  return (
    <SourcesOptionMenuWrapper>
      <KeyvalSearchInput />
      <DropdownWrapper>
        <KeyvalText size={14}>{"Namespace"}</KeyvalText>
        <KeyvalDropDown />
      </DropdownWrapper>
    </SourcesOptionMenuWrapper>
  );
}
