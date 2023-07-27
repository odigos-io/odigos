"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { DestinationContainerWrapper } from "./destination.styled";
import { NewDestinationFlow } from "./new.destination.flow";
import { UpdateDestinationFlow } from "./update.destination.flow";

export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const [displayNewDestination, setDisplayNewDestination] =
    useState<boolean>(false);

  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  function handleAddNewDestinationClick() {
    setDisplayNewDestination(true);
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  if (displayNewDestination) {
    return (
      <NewDestinationFlow
        onBackClick={() => {
          refetch();
          setDisplayNewDestination(false);
        }}
      />
    );
  }

  return (
    <DestinationContainerWrapper>
      {selectedDestination ? (
        <UpdateDestinationFlow
          selectedDestination={selectedDestination}
          setSelectedDestination={setSelectedDestination}
        />
      ) : (
        <>
          <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
          <DestinationsManagedList
            data={destinationList}
            onItemClick={setSelectedDestination}
            onMenuButtonClick={handleAddNewDestinationClick}
          />
        </>
      )}
    </DestinationContainerWrapper>
  );
}
