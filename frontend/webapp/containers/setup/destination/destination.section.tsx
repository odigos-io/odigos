import { DestinationList, DestinationOptionMenu } from "@/components/setup";
import React, { useState } from "react";
import { DestinationListContainer } from "./destination.section.styled";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";

export function DestinationSection({ sectionData, setSectionData }: any) {
  const [searchFilter, setSearchFilter] = useState<string>("");
  const [dropdownData, setDropdownData] = useState<any>(null);
  const [monitoringOption, setMonitoringOption] =
    useState<any>(MONITORING_OPTIONS);

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  function filterDataByTextQuery() {
    const filteredData = data?.categories.map((category: any) => {
      const items = category.items.filter((item: any) => {
        const displayType = item.display_type.toLowerCase();
        const searchFilterLower = searchFilter.toLowerCase();

        return displayType.includes(searchFilterLower);
      });

      return {
        ...category,
        items,
      };
    });

    return filteredData;
  }

  function filterDataByMonitorsOption(data: any) {
    const selectedMonitors = monitoringOption
      .filter((monitor: any) => monitor.tapped)
      .map((monitor: any) => monitor.title.toLowerCase());

    if (selectedMonitors.length === 3) return data;

    const filteredData: any[] = [];

    data?.forEach((category: any) => {
      const supportedItems: any[] = [];
      category.items.filter((item: any) => {
        const supportedSignals: any[] = [];
        for (const monitor in item.supported_signals) {
          if (item.supported_signals[monitor].supported) {
            supportedSignals.push(monitor);
          }
        }

        const found = selectedMonitors.some((r) =>
          supportedSignals.includes(r)
        );

        if (found) {
          supportedItems.push(item);
        }
      });

      filteredData.push({
        items: supportedItems,
        name: category.name,
      });
    });

    return filteredData;
  }

  function renderDestinationLists() {
    let list = searchFilter ? filterDataByTextQuery() : data?.categories; //TODO change to real data (sectionData)
    list = filterDataByMonitorsOption(list);
    return list?.map((category: any, index: number) => {
      const displayItem =
        dropdownData?.label === category.name || dropdownData?.label === "All";

      return (
        displayItem && (
          <DestinationList
            key={index}
            data={category}
            sectionData={sectionData}
            onItemClick={(item: any) => setSectionData(item)}
          />
        )
      );
    });
  }

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <>
      <DestinationOptionMenu
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
        setDropdownData={setDropdownData}
        monitoringOption={monitoringOption}
        setMonitoringOption={setMonitoringOption}
        data={data.categories}
      />
      <DestinationListContainer>
        {renderDestinationLists()}
      </DestinationListContainer>
    </>
  );
}
