import { DestinationList, DestinationOptionMenu } from "@/components/setup";
import React, { useState } from "react";
import { DestinationListContainer } from "./destination.section.styled";
import { QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";

export function DestinationSection({ sectionData, setSectionData }: any) {
  const [searchFilter, setSearchFilter] = useState<string>("");
  const [dropdownData, setDropdownData] = useState<any>(null);

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  function filterData() {
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

  function renderDestinationLists() {
    const list = searchFilter ? filterData() : data?.categories; //TODO change to real data (sectionData)

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
        data={data.categories}
      />
      <DestinationListContainer>
        {renderDestinationLists()}
      </DestinationListContainer>
    </>
  );
}
