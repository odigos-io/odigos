"use client";
import React, { useEffect, useState } from "react";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useNotification } from "@/hooks";
import { useSearchParams } from "next/navigation";
import UpdateDestinationFlow from "@/containers/overview/destination/update.destination.flow";
import { getDestinations } from "@/services";
import { useQuery } from "react-query";
import { KeyvalLoader } from "@/design.system";

const DEST = "dest";

export default function ManageDestinationPage() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const { show, Notification } = useNotification();

  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  const searchParams = useSearchParams();

  useEffect(onPageLoad, [searchParams, destinationList]);

  function onPageLoad() {
    const search = searchParams.get(DEST);
    const currentDestination = destinationList?.filter(
      ({ id }) => id === search
    );
    if (currentDestination?.length) {
      setSelectedDestination(currentDestination[0]);
    }
  }

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

  if (destinationLoading || !selectedDestination) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <UpdateDestinationFlow
        selectedDestination={selectedDestination}
        onSuccess={onSuccess}
        onError={onError}
      />
      <Notification />
    </>
  );
}
