import React, { useEffect, useState } from "react";
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
import { SelectedCounter } from "@/design.system/selected.counter/selected.counter";

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
        <KeyvalText size={14}>{"Namespace"}</KeyvalText>
        <KeyvalDropDown
          data={data}
          onChange={handleDropDownChange}
          childComp={<SelectedCounter total={10} selected={5} />}
        />
      </DropdownWrapper>

      <KeyvalSwitch
        label="Select All"
        toggle={toggle}
        handleToggleChange={handleToggleChange}
      />
      <KeyvalCheckbox
        label="Apply for any future apps"
        value={checked}
        onChange={() => onFutureApplyChange(!checked)}
      />
    </SourcesOptionMenuWrapper>
  );
}

const ChildComp: React.FC = () => <h2>This is a child component</h2>;
