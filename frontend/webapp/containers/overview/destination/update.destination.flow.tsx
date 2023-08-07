"use client";
import React, { useEffect, useMemo, useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, updateDestination } from "@/services";
import { ManageDestination } from "@/components/overview";
import { deleteDestination, getDestinations } from "@/services/destinations";
import { ManageDestinationWrapper } from "./destination.styled";
import { useRouter, useSearchParams } from "next/navigation";
import { useNotification } from "@/hooks";
const DEST = "dest";

export function UpdateDestinationFlow() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);

  const router = useRouter();
  const searchParams = useSearchParams();
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

  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  const { mutate: handleUpdateDestination } = useMutation((body) =>
    updateDestination(body, selectedDestination?.id)
  );

  const { mutate: handleDeleteDestination } = useMutation((body) =>
    deleteDestination(selectedDestination?.id)
  );

  useEffect(onPageLoad, [searchParams, destinationList]);

  function onDelete() {
    handleDeleteDestination(selectedDestination.id, {
      onSuccess: () => router.push(`${ROUTES.DESTINATIONS}?status=deleted`),
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

  return destinationTypeLoading ? (
    <KeyvalLoader />
  ) : (
    <ManageDestinationWrapper>
      <ManageDestination
        onBackClick={() => router.back()}
        destinationType={destinationType}
        selectedDestination={manageData}
        onSubmit={onSubmit}
        onDelete={onDelete}
      />
      <Notification />
    </ManageDestinationWrapper>
  );
}
