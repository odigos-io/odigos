"use client";
import React, { useMemo } from "react";
import { KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, updateDestination } from "@/services";
import { ManageDestination } from "@/components/overview";
import { deleteDestination } from "@/services/destinations";
import { useNotification } from "@/hooks";

export function UpdateDestinationFlow({
  selectedDestination,
  setSelectedDestination,
  onSuccess,
  onError,
}) {
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

  function onDelete() {
    handleDeleteDestination(selectedDestination.id, {
      onSuccess: () => onSuccess(OVERVIEW.DESTINATION_DELETED_SUCCESS),
      onError,
    });
  }

  function onSubmit(updatedDestination) {
    const newDestinations = {
      ...updatedDestination,
      //   type: selectedDestination.type,
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
    </>
  );
}
