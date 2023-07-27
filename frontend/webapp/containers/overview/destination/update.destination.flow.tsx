"use client";
import React, { useMemo } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, updateDestination } from "@/services";
import { ManageDestination } from "@/components/overview";
import { useNotification } from "@/hooks";
import { deleteDestination } from "@/services/destinations";

export function UpdateDestinationFlow({
  selectedDestination,
  setSelectedDestination,
}) {
  const { show, Notification } = useNotification();

  const manageData = useMemo(() => {
    return {
      ...selectedDestination,
      ...selectedDestination?.destination_type,
    };
  }, [selectedDestination]);

  const { isLoading: destinationTypeLoading, data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, selectedDestination?.type],
    () => getDestination(selectedDestination?.type),
    {
      enabled: !!selectedDestination,
    }
  );

  const { mutate: handleUpdateDestination } = useMutation((body) =>
    updateDestination(body, selectedDestination?.id)
  );

  const { mutate: handleDeleteDestination } = useMutation((body) =>
    deleteDestination(selectedDestination?.id)
  );

  function onBackClick() {
    setSelectedDestination(null);
  }

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    setSelectedDestination(null);
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

  function onDelete() {
    handleDeleteDestination(selectedDestination.id, {
      onSuccess: () => onSuccess(OVERVIEW.DESTINATION_DELETED_SUCCESS),
      onError,
    });
  }

  function onSubmit(updatedDestination) {
    const newDestinations = {
      ...updatedDestination,
      type: selectedDestination.type,
    };

    handleUpdateDestination(newDestinations, {
      onSuccess,
      onError,
    });
  }

  if (destinationTypeLoading) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <ManageDestination
        onBackClick={onBackClick}
        destinationType={destinationType}
        selectedDestination={manageData}
        onSubmit={onSubmit}
        onDelete={onDelete}
      />
      <Notification />
    </>
  );
}
