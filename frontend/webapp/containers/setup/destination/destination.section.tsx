import { DestinationList, DestinationOptionMenu } from "@/components/setup";
import React, { useEffect, useMemo, useState } from "react";
import { DestinationListContainer } from "./destination.section.styled";

const FAKE = {
  categories: [
    {
      name: "Managed",
      items: [
        {
          type: "newrelic",
          display_type: "New Relic",
          image_url: "https://s3.amazonaws.com/keyval-dev/newrelic.jpg",
          supported_signals: {
            traces: {
              supported: true,
            },
            metrics: {
              supported: true,
            },
            logs: {
              supported: false,
            },
          },
        },
      ],
    },
    {
      name: "Self hosted",
      items: [
        {
          type: "jaeger",
          display_type: "Jaeger",
          image_url: "https://s3.amazonaws.com/keyval-dev/jaeger.jpg",
          supported_signals: {
            traces: {
              supported: true,
            },
            metrics: {
              supported: false,
            },
            logs: {
              supported: false,
            },
          },
        },
      ],
    },
  ],
};

export function DestinationSection({ sectionData, setSectionData }: any) {
  const [searchFilter, setSearchFilter] = useState<string>("");

  function filterData() {
    const filteredData = FAKE.categories.map((category: any) => {
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
    const list = searchFilter ? filterData() : FAKE.categories; //TODO change to real data (sectionData)

    return list?.map((category: any, index: number) => (
      <DestinationList
        key={index}
        data={category}
        onItemClick={(item) => setSectionData(item)}
      />
    ));
  }

  return (
    <>
      <DestinationOptionMenu
        searchFilter={searchFilter}
        setSearchFilter={setSearchFilter}
      />
      <DestinationListContainer>
        {renderDestinationLists()}
      </DestinationListContainer>
    </>
  );
}
