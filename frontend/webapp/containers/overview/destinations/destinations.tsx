"use client";
import React, { useEffect } from "react";
import { KeyvalLoader } from "@/design.system";
import {
  NOTIFICATION,
  OVERVIEW,
  PARAMS,
  QUERIES,
  ROUTES,
} from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { useRouter, useSearchParams } from "next/navigation";
import { useNotification } from "@/hooks";

export function DestinationContainer() {
  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch: refetchDestinations,
  } = useQuery([QUERIES.API_DESTINATIONS], getDestinations);

  const searchParams = useSearchParams();
  const { show, Notification } = useNotification();

  const router = useRouter();

  useEffect(onPageLoad, [searchParams, destinationList]);

  function getMessage(status: string) {
    switch (status) {
      case PARAMS.DELETED:
        return OVERVIEW.DESTINATION_DELETED_SUCCESS;
      case PARAMS.CREATED:
        return OVERVIEW.DESTINATION_CREATED_SUCCESS;
      case PARAMS.UPDATED:
        return OVERVIEW.DESTINATION_UPDATE_SUCCESS;
      default:
        return "";
    }
  }

  function onPageLoad() {
    const status = searchParams.get(PARAMS.STATUS);
    if (status) {
      refetchDestinations();
      show({
        type: NOTIFICATION.SUCCESS,
        message: getMessage(status),
      });
      router.replace(ROUTES.DESTINATIONS);
    }
  }

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <DestinationsManagedList
        data={destinationList}
        onItemClick={({ id }) =>
          router.push(`${ROUTES.UPDATE_DESTINATION}${id}`)
        }
        onMenuButtonClick={() => router.push(ROUTES.CREATE_DESTINATION)}
      />
      <Notification />
    </>
  );
}
