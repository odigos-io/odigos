"use client";
import React from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, updateDestination } from "@/services";
import { ManageDestination } from "@/components/overview";
import { useNotification } from "@/hooks";

export function UpdateDestinationFlow({
  selectedDestination,
  setSelectedDestination,
}) {
  const { show, Notification } = useNotification();

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

  function onBackClick() {
    setSelectedDestination(null);
  }

  function onSubmit(updatedDestination) {
    const newDestinations = {
      ...updatedDestination,
      type: selectedDestination.type,
    };

    function onSuccess() {
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

  if (destinationTypeLoading) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <ManageDestination
        onBackClick={onBackClick}
        destinationType={destinationType}
        selectedDestination={getSelectionData()}
        onSubmit={onSubmit}
      />
      <Notification />
    </>
  );
}
