"use client";
import React, { useEffect } from "react";
import { KeyvalLoader } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { useRouter, useSearchParams } from "next/navigation";
import { useNotification } from "@/hooks";

export function DestinationContainer() {
  const { isLoading: destinationLoading, data: destinationList } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const searchParams = useSearchParams();
  const { show, Notification } = useNotification();

  const router = useRouter();

  useEffect(onPageLoad, [searchParams, destinationList]);

  function onPageLoad() {
    const status = searchParams.get("status");
    if (status === "deleted") {
      show({
        type: NOTIFICATION.SUCCESS,
        message: OVERVIEW.DESTINATION_DELETED_SUCCESS,
      });
      router.push(ROUTES.DESTINATIONS);
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
