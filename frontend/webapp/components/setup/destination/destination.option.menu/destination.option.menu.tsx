import React, { useMemo, useState } from "react";
import {
  DropdownWrapper,
  SourcesOptionMenuWrapper,
  TapsWrapper,
} from "./destination.option.menu.styled";
import {
  KeyvalDropDown,
  KeyvalSearchInput,
  KeyvalTap,
  KeyvalText,
} from "@/design.system";
import { SETUP } from "@/utils/constants";
import Logs from "@/assets/icons/logs-grey.svg";
import LogsFocus from "@/assets/icons/logs-blue.svg";
import Metrics from "@/assets/icons/chart-line-grey.svg";
import MetricsFocus from "@/assets/icons/chart-line-blue.svg";
import Traces from "@/assets/icons/tree-structure-grey.svg";
import TracesFocus from "@/assets/icons/tree-structure-blue.svg";
import { TapList } from "../tap.list/tap.list";

const MONITORING_OPTIONS = [
  {
    id: "1",
    icons: {
      notFocus: () => Logs(),
      focus: () => LogsFocus(),
    },
    title: "Logs",
    tapped: false,
  },
  {
    id: "1",
    icons: {
      notFocus: () => Metrics(),
      focus: () => MetricsFocus(),
    },
    title: "Metrics",
    tapped: true,
  },
  {
    id: "1",
    icons: {
      notFocus: () => Traces(),
      focus: () => TracesFocus(),
    },
    title: "Traces",
    tapped: false,
  },
];

export function DestinationOptionMenu({
  setDropdownData,
  data,
  searchFilter,
  setSearchFilter,
}: any) {
  const [monitoringOption, setMonitoringOption] =
    useState<any>(MONITORING_OPTIONS);

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
        <TapList list={monitoringOption} />
      </TapsWrapper>
    </SourcesOptionMenuWrapper>
  );
}
