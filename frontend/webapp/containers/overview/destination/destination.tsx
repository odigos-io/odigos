"use client";
import React from "react";
import { KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import { OverviewHeader, DestinationsManagedList } from "@/components/overview";
import { useRouter } from "next/navigation";

export function DestinationContainer() {
  const { isLoading: destinationLoading, data: destinationList } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const router = useRouter();

  if (destinationLoading) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <DestinationsManagedList
        data={destinationList}
        onItemClick={({ id }) => router.push(`destinations/manage?dest=${id}`)}
        onMenuButtonClick={() => router.push("destinations/create")}
      />
    </>
  );
}
