import React, { useEffect, useState } from "react";
import { useQuery } from "react-query";
import { NOTIFICATION, QUERIES, SETUP } from "@/utils/constants";
import { MONITORING_OPTIONS } from "@/components/setup/destination/utils";
import { DestinationList, DestinationOptionMenu } from "@/components/setup";
import Empty from "@/assets/images/empty-list.svg";
import {
  DestinationListContainer,
  EmptyListWrapper,
  LoaderWrapper,
} from "./destination.section.styled";
import {
  filterDataByMonitorsOption,
  filterDataByTextQuery,
  isDestinationListEmpty,
  sortDestinationList,
} from "./utils";
import { KeyvalLoader } from "@/design.system";
import { useNotification } from "@/hooks";
import { getDestinationsTypes } from "@/services";

type DestinationSectionProps = {
  sectionData?: any;
  setSectionData: (data: any) => void;
};

export function DestinationSection({
  sectionData,
  setSectionData,
}: DestinationSectionProps) {
  const [searchFilter, setSearchFilter] = useState<string>("");
  const [dropdownData, setDropdownData] = useState<any>(null);
  const { show, Notification } = useNotification();
  const [monitoringOption, setMonitoringOption] =
    useState<any>(MONITORING_OPTIONS);

  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  useEffect(() => {
    isError &&
      show({
        type: NOTIFICATION.ERROR,
        message: error,
      });
  }, [isError]);

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
            sectionData={sectionData}
            key={category.name}
            data={category}
            onItemClick={(item: any) => setSectionData(item)}
          />
        )
      );
    });
  }

  if (isLoading) {
    return (
      <LoaderWrapper>
        <KeyvalLoader />
      </LoaderWrapper>
    );
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
      <Notification />
    </>
  );
}
