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

  function renderDestinationLists() {
    return FAKE.categories.map((category: any, index: number) => (
      <DestinationList key={index} data={category} />
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
