import React from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
} from "./sources.option.menu.styled";
import { KeyvalDropDown, KeyvalSearchInput, KeyvalText } from "@/design.system";
const DATA = [
  { id: 1, label: "Istanbul, TR (AHL)" },
  { id: 2, label: "Paris, FR (CDG)" },
];

export function SourcesOptionMenu() {
  return (
    <SourcesOptionMenuWrapper>
      <KeyvalSearchInput />
      <DropdownWrapper>
        <KeyvalText size={14}>{"Namespace"}</KeyvalText>
        <KeyvalDropDown data={DATA} />
      </DropdownWrapper>
    </SourcesOptionMenuWrapper>
  );
}
