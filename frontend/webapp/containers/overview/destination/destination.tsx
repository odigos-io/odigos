"use client";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { KeyvalButton, KeyvalLoader, KeyvalText } from "@/design.system";
import { OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations, getDestination } from "@/services";
import DestinationsManagedList from "@/components/overview/destination/destination.list/destinations.managed.list";
import { MenuWrapper } from "./destination.styled";
import { Plus } from "@/assets/icons/overview";
import { useRouter } from "next/navigation";
import { ManageDestination } from "@/components/overview/destination/manage.destination/manage.destination";
import { OverviewHeader } from "@/components/overview";
export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(false);

  const router = useRouter();

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { data: destination } = useQuery(
    [QUERIES.API_DESTINATION_TYPE],
    () => getDestination(selectedDestination?.type),
    {
      enabled: !!selectedDestination,
    }
  );

  useEffect(() => {
    console.log({ selectedDestination });
  }, [selectedDestination]);

  function handleAddNewDestinationClick() {
    router.push(`${ROUTES.SETUP}?${"state=destinations"}`);
  }

  if (isLoading) {
    return <KeyvalLoader />;
  }

  return (
    <div
      style={{
        height: "100%",
        width: "100%",
        overflowY: "scroll",
      }}
    >
      {destination && selectedDestination ? (
        <ManageDestination
          onBackClick={() => setSelectedDestination(false)}
          data={destination}
          supportedSignals={
            selectedDestination?.destination_type?.supported_signals
          }
        />
      ) : (
        <>
          <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
          <DestinationsManagedList
            data={data}
            onItemClick={setSelectedDestination}
            onMenuButtonClick={handleAddNewDestinationClick}
          />
        </>
      )}
    </div>
  );
}
