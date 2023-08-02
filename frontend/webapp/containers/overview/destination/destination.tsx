"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { DestinationContainerWrapper } from "./destination.styled";
import { UpdateDestinationFlow } from "./update.destination.flow";
import { useNotification } from "@/hooks";
import { useRouter } from "next/navigation";

export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);

  const { show, Notification } = useNotification();
  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  const router = useRouter();

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    refetch();
    setSelectedDestination(null);
    router.push("destinations");
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

  function renderUpdateDestinationFlow() {
    return (
      <UpdateDestinationFlow
        selectedDestination={selectedDestination}
        setSelectedDestination={setSelectedDestination}
        onSuccess={onSuccess}
        onError={onError}
      />
    );
  }

  function renderDestinationList() {
    return (
      <>
        <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
        <DestinationsManagedList
          data={destinationList}
          onItemClick={setSelectedDestination}
          onMenuButtonClick={() => router.push("destinations/create")}
        />
      </>
    );
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <DestinationContainerWrapper>
      {selectedDestination
        ? renderUpdateDestinationFlow()
        : renderDestinationList()}
      <Notification />
    </DestinationContainerWrapper>
  );
}
