"use client";
import React, { useState } from "react";
import { KeyvalLoader } from "@/design.system";
import { OVERVIEW, QUERIES, ROUTES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestinations, getDestination } from "@/services";
import DestinationsManagedList from "@/components/overview/destination/destination.list/destinations.managed.list";
import { useRouter } from "next/navigation";
import { ManageDestination } from "@/components/overview/destination/manage.destination/manage.destination";
import { OverviewHeader } from "@/components/overview";
import { updateDestination } from "@/services/destinations";

export function DestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(false);
  const router = useRouter();

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { isLoading: loading, data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, selectedDestination?.type],
    () => getDestination(selectedDestination?.type),
    {
      enabled: !!selectedDestination,
    }
  );

  const { mutate } = useMutation((body) => updateDestination(body));

  function handleAddNewDestinationClick() {
    router.push(`${ROUTES.SETUP}?${"state=destinations"}`);
  }

  function onBackClick() {
    setSelectedDestination(false);
  }

  function onSubmit(data) {
    const newDestinations = {
      ...data,
      type: selectedDestination.type,
    };

    mutate(newDestinations, {
      onSuccess: () => console.log("onSuccess"),
      onError: () => console.log("onError"),
    });

    console.log("newDestinations", newDestinations);
  }

  if (isLoading || loading) {
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
      {destinationType && selectedDestination ? (
        <ManageDestination
          onBackClick={onBackClick}
          destinationType={destinationType}
          selectedDestination={selectedDestination}
          onSubmit={onSubmit}
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
