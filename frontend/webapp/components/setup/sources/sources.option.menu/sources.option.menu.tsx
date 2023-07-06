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

export function SourcesOptionMenu({
  setCurrentItem,
  data,
  searchFilter,
  setSearchFilter,
}: any) {
  const [checked, setChecked] = useState(false);
  const [toggle, setToggle] = useState(false);
  const handleToggleChange = () => {
    setToggle(!toggle);
  };

  function handleDropDownChange(item: any) {
    setCurrentItem({ id: item?.id, name: item.label });
  }

  return (
    <SourcesOptionMenuWrapper>
      <KeyvalSearchInput
        value={searchFilter}
        onChange={(e) => setSearchFilter(e.target.value)}
      />

      <DropdownWrapper>
        <KeyvalText size={14}>{"Namespace"}</KeyvalText>
        <KeyvalDropDown data={data} onChange={handleDropDownChange} />
      </DropdownWrapper>

      <KeyvalSwitch
        label="Select All"
        toggle={toggle}
        handleToggleChange={handleToggleChange}
      />
      <KeyvalCheckbox
        label="Apply for any future apps"
        value={checked}
        onChange={() => setChecked(!checked)}
      />
    </SourcesOptionMenuWrapper>
  );
}
