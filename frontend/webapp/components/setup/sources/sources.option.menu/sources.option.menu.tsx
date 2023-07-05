import React, { useState } from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
} from "./sources.option.menu.styled";
import {
  KeyvalCheckbox,
  KeyvalDropDown,
  KeyvalSearchInput,
  KeyvalSwitch,
  KeyvalText,
} from "@/design.system";
const DATA = [
  { id: 1, label: "Istanbul, TR (AHL)" },
  { id: 2, label: "Paris, FR (CDG)" },
];

export function SourcesOptionMenu() {
  const [checked, setChecked] = useState(false);
  const [toggle, setToggle] = useState(false);
  const handleToggleChange = () => {
    setToggle(!toggle);
  };

  return (
    <SourcesOptionMenuWrapper>
      <KeyvalSearchInput />

      <DropdownWrapper>
        <KeyvalText size={14}>{"Namespace"}</KeyvalText>
        <KeyvalDropDown data={DATA} />
      </DropdownWrapper>

      <KeyvalSwitch toggle={toggle} handleToggleChange={handleToggleChange} />
      <KeyvalCheckbox value={checked} onChange={() => setChecked(!checked)} />
    </SourcesOptionMenuWrapper>
  );
}
