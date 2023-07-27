"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { DestinationContainerWrapper } from "./destination.styled";
import { NewDestinationFlow } from "./new.destination.flow";
import { UpdateDestinationFlow } from "./update.destination.flow";
import { useNotification } from "@/hooks";

export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const [displayNewDestination, setDisplayNewDestination] =
    useState<boolean>(false);
  const { show, Notification } = useNotification();
  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    refetch();
    setSelectedDestination(null);
    setDisplayNewDestination(false);
    show({
      type: NOTIFICATION.SUCCESS,
      message,
    });
  }

  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <DestinationContainerWrapper>
      {displayNewDestination ? (
        <NewDestinationFlow
          onSuccess={onSuccess}
          onError={onError}
          onBackClick={() => {
            setDisplayNewDestination(false);
          }}
        />
      ) : selectedDestination ? (
        <UpdateDestinationFlow
          selectedDestination={selectedDestination}
          setSelectedDestination={setSelectedDestination}
          onSuccess={onSuccess}
          onError={onError}
        />
      ) : (
        <>
          <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
          <DestinationsManagedList
            data={destinationList}
            onItemClick={setSelectedDestination}
            onMenuButtonClick={() => setDisplayNewDestination(true)}
          />
        </>
      )}
      <Notification />
    </DestinationContainerWrapper>
  );
}
