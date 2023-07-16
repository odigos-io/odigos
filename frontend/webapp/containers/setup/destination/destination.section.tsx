import React, { useState } from "react";
import { useQuery } from "react-query";
import { QUERIES, SETUP } from "@/utils/constants";
import { getDestinations } from "@/services/setup";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";
import { DestinationList, DestinationOptionMenu } from "@/components/setup";
import Empty from "@/assets/images/empty-list.svg";
import {
  DestinationListContainer,
  EmptyListWrapper,
} from "./destination.section.styled";
import {
  filterDataByMonitorsOption,
  filterDataByTextQuery,
  isDestinationListEmpty,
  sortDestinationList,
} from "./utils";

type DestinationSectionProps = {
  sectionData: any;
  setSectionData: (data: any) => void;
};

export function DestinationSection({
  sectionData,
  setSectionData,
}: DestinationSectionProps) {
  const [searchFilter, setSearchFilter] = useState<string>("");
  const [dropdownData, setDropdownData] = useState<any>(null);
  const [monitoringOption, setMonitoringOption] =
    useState<any>(MONITORING_OPTIONS);

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  function renderDestinationLists() {
    sortDestinationList(data);
    const list = filterDataByMonitorsOption(
      filterDataByTextQuery(data, searchFilter),
      monitoringOption
    );

    if (isDestinationListEmpty(list, dropdownData?.id)) {
      return (
        <EmptyListWrapper>
          <Empty />
        </EmptyListWrapper>
      );
    }

    return list?.map((category: any) => {
      const displayItem =
        dropdownData?.label === category.name ||
        dropdownData?.label === SETUP.ALL;

      return (
        displayItem && (
          <DestinationList
            key={category.name}
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
        data={data?.categories}
      />
      {data && (
        <DestinationListContainer>
          {renderDestinationLists()}
        </DestinationListContainer>
      )}
    </>
  );
}
