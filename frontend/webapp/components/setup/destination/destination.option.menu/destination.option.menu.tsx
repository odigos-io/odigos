import React, { useState } from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
} from "./destination.option.menu.styled";
import { KeyvalDropDown, KeyvalSearchInput, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";

export function DestinationOptionMenu({
  setCurrentItem,
  data,
  searchFilter,
  setSearchFilter,
}: any) {
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
        <KeyvalText size={14}>{SETUP.MENU.TYPE}</KeyvalText>
        <KeyvalDropDown
          width={180}
          data={data}
          onChange={handleDropDownChange}
        />
      </DropdownWrapper>
    </SourcesOptionMenuWrapper>
  );
}
