"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
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

export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const router = useRouter();
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
    router.push(ROUTES.NEW_DESTINATION);
  }

  function onBackClick() {
    setSelectedDestination(null);
  }

  function onSubmit(updatedDestination: any) {
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

    function onError({ response }: any) {
      const message = response?.data?.message || "";
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

  if (destinationLoading || destinationTypeLoading) {
    return <KeyvalLoader />;
  }

  return (
    <DestinationContainerWrapper>
      {destinationType && selectedDestination ? (
        <ManageDestination
          onBackClick={onBackClick}
          destinationType={destinationType}
          selectedDestination={selectedDestination}
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
