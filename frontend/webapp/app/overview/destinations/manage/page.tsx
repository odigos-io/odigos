"use client";
import React, { useEffect, useState } from "react";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useNotification } from "@/hooks";
import { useSearchParams } from "next/navigation";
import UpdateDestinationFlow from "@/containers/overview/destination/update.destination.flow";
import { getDestinations } from "@/services";
import { useQuery } from "react-query";

export default function ManageDestinationPage() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);

  const { show, Notification } = useNotification();
  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  const searchParams = useSearchParams();

  useEffect(() => {
    const search = searchParams.get("dest");
    const currentDestination = destinationList?.filter(
      (item) => item?.id === search
    );
    if (currentDestination?.length) {
      setSelectedDestination(currentDestination[0]);
    }
  }, [searchParams, destinationList]);

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    refetch();
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
    return;
  }

  return (
    selectedDestination && (
      <>
        <UpdateDestinationFlow
          selectedDestination={selectedDestination}
          onSuccess={onSuccess}
          onError={onError}
        />
        <Notification />
      </>
    )
  );
}
