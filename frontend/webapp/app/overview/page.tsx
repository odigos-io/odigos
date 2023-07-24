"use client";
import React, { useEffect } from "react";
import { KeyvalFlow, KeyvalLoader } from "@/design.system";
import { OverviewHeader } from "@/components/overview";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";

export default function OverviewPage() {
  const { isLoading, data: destinations } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { data: sources } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  if (isLoading) {
    return <KeyvalLoader />;
  }

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.OVERVIEW} />
      <KeyvalFlow destinations={destinations} sources={sources} />
    </>
  );
}
