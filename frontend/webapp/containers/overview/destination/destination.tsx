"use client";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { KeyvalButton, KeyvalLoader, KeyvalText } from "@/design.system";
import { QUERIES, ROUTES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services";
import DestinationsManagedList from "@/components/overview/destination/destination.list/destinations.managed.list";
import { MenuWrapper } from "./destination.styled";
import { Plus } from "@/assets/icons/overview";
import { useRouter } from "next/navigation";
export function DestinationContainer() {
  const router = useRouter();
  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  useEffect(() => {
    console.log({ data });
  }, [data]);

  function handleAddNewDestinationClick() {
    console.log("Add new destination");
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
      <MenuWrapper>
        <KeyvalText>{`${data.length} Applications`}</KeyvalText>
        <KeyvalButton
          onClick={handleAddNewDestinationClick}
          style={{ gap: 10, width: 224, height: 40 }}
        >
          <Plus />
          <KeyvalText size={16} weight={700} color="#0A1824">
            {"Add New Destination"}
          </KeyvalText>
        </KeyvalButton>
      </MenuWrapper>
      <DestinationsManagedList data={data} />
    </div>
  );
}
