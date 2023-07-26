"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { useRouter } from "next/navigation";
import { getDestinations, getDestination, updateDestination } from "@/services";
import {
  OverviewHeader,
  ManageDestination,
  DestinationsManagedList,
} from "@/components/overview";
import { DestinationContainerWrapper } from "./destination.styled";
import { useNotification } from "@/hooks";
import { NewDestinationFlow } from "./new.destination.flow";

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

  const { isLoading: destinationTypeLoading, data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, selectedDestination?.type],
    () => getDestination(selectedDestination?.type),
    {
      enabled: !!selectedDestination,
    }
  );

  const { mutate } = useMutation((body) =>
    updateDestination(body, selectedDestination?.id)
  );

  function handleAddNewDestinationClick() {
    setDisplayNewDestination(true);
  }

  function onBackClick() {
    setSelectedDestination(null);
  }

  function onSubmit(updatedDestination) {
    const newDestinations = {
      ...updatedDestination,
      type: selectedDestination.type,
    };

    function onSuccess() {
      refetch();
      setSelectedDestination(null);
      show({
        type: NOTIFICATION.SUCCESS,
        message: OVERVIEW.DESTINATION_UPDATE_SUCCESS,
      });
    }

    function onError({ response }) {
      const message = response?.data?.message;
      show({
        type: NOTIFICATION.ERROR,
        message,
      });
    }

    mutate(newDestinations, {
      onSuccess,
      onError,
    });
  }

  function getSelectionData() {
    const newDestinations = {
      ...selectedDestination,
      ...selectedDestination?.destination_type,
    };

    return newDestinations;
  }

  if (destinationLoading || destinationTypeLoading) {
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
      {destinationType && selectedDestination ? (
        <ManageDestination
          onBackClick={onBackClick}
          destinationType={destinationType}
          selectedDestination={getSelectionData()}
          onSubmit={onSubmit}
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
      <Notification />
    </DestinationContainerWrapper>
  );
}
