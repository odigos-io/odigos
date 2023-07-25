"use client";
import { OverviewHeader } from "@/components/overview";
import { DestinationContainer } from "@/containers/overview";
import { OVERVIEW } from "@/utils/constants";
import React from "react";

export default function DestinationDashboardPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <DestinationContainer />
    </>
  );
}
