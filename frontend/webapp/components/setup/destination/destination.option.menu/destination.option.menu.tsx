import React, { useMemo, useState } from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
  TapsWrapper,
} from "./destination.option.menu.styled";
import { KeyvalDropDown, KeyvalSearchInput, KeyvalText } from "@/design.system";
import { SETUP } from "@/utils/constants";
import { TapList } from "../tap.list/tap.list";

export function DestinationOptionMenu({
  setDropdownData,
  data,
  searchFilter,
  setSearchFilter,
  monitoringOption,
  setMonitoringOption,
}: any) {
  const dropdownData = useMemo(() => {
    let dropdownList = data?.map(({ name }: any) => {
      return {
        id: name,
        label: name,
      };
    });

    dropdownList.unshift({ id: "all", label: "All" });
    setDropdownData(dropdownList[0]);
    return dropdownList;
  }, [data]);

  function handleDropDownChange(item: any) {
    setDropdownData({ id: item?.id, label: item.label });
  }

  function handleTapClick(id: any) {
    const currentMonitorIndex = monitoringOption.findIndex(
      (monitor) => monitor.id === id
    );

    const newMonitor = {
      ...monitoringOption[currentMonitorIndex],
      tapped: !monitoringOption[currentMonitorIndex].tapped,
    };

    const newMonitoringOption = [...monitoringOption];
    newMonitoringOption[currentMonitorIndex] = newMonitor;

    setMonitoringOption(newMonitoringOption);
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
          data={dropdownData}
          onChange={handleDropDownChange}
        />
      </DropdownWrapper>
      <TapsWrapper>
        <KeyvalText size={14}>{SETUP.MENU.MONITORING}</KeyvalText>
        <TapList list={monitoringOption} onClick={handleTapClick} />
      </TapsWrapper>
    </SourcesOptionMenuWrapper>
  );
}
