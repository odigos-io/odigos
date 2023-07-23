import React, { useEffect, useState } from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
  CheckboxWrapper,
  SwitcherWrapper,
} from "./sources.option.menu.styled";
import {
  KeyvalCheckbox,
  KeyvalDropDown,
  KeyvalSearchInput,
  KeyvalSwitch,
  KeyvalText,
  KeyvalTooltip,
} from "@/design.system";
import { SETUP } from "@/utils/constants";

export function SourcesOptionMenu({
  setCurrentItem,
  data,
  searchFilter,
  setSearchFilter,
  onSelectAllChange,
  selectedApplications,
  currentNamespace,
  onFutureApplyChange,
}: any) {
  const [checked, setChecked] = useState(false);
  const [toggle, setToggle] = useState(false);

  useEffect(() => {
    setToggle(selectedApplications[currentNamespace?.name]?.selected_all);
    setChecked(selectedApplications[currentNamespace?.name]?.future_selected);
  }, [currentNamespace, selectedApplications]);

  const handleToggleChange = () => {
    onSelectAllChange(!toggle);
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
        <KeyvalText size={14}>{SETUP.MENU.NAMESPACES}</KeyvalText>
        <KeyvalDropDown
          value={data[0]}
          data={data}
          onChange={handleDropDownChange}
        />
      </DropdownWrapper>
      <SwitcherWrapper>
        <KeyvalSwitch
          label={SETUP.MENU.SELECT_ALL}
          toggle={toggle}
          handleToggleChange={handleToggleChange}
        />
      </SwitcherWrapper>
      <CheckboxWrapper>
        <KeyvalCheckbox
          label={SETUP.MENU.FUTURE_APPLY}
          value={checked}
          onChange={() => onFutureApplyChange(!checked)}
          disabled={!toggle}
        />
        <KeyvalTooltip content={SETUP.MENU.TOOLTIP} />
      </CheckboxWrapper>
    </SourcesOptionMenuWrapper>
  );
}
