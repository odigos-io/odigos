"use client";
import React, { useEffect } from "react";
import { KeyvalFlow } from "@/design.system";
import { OverviewHeader } from "@/components/overview";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useQuery } from "react-query";
import { getDestinations } from "@/services/setup";
export default function OverviewPage() {
  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.OVERVIEW} />
      <KeyvalFlow destinations={data} />
    </>
  );
}
