"use client";
import React from "react";
import { OVERVIEW } from "@/utils/constants";
import { useMutation } from "react-query";
import { setDestination } from "@/services";
import { OverviewHeader } from "@/components/overview";
import { useSectionData } from "@/hooks";
import { DestinationSection } from "@/containers/setup/destination/destination.section";
import { NewDestinationContainer } from "./destination.styled";
import { useRouter } from "next/navigation";

export function NewDestinationFlow() {
  const { sectionData, setSectionData } = useSectionData(null);
  const router = useRouter();

  function handleBackPress() {
    router.back();
  }

  function renderSelectNewDestination() {
    return (
      <DestinationSection
        sectionData={sectionData}
        setSectionData={(data) => {
          router.push(`/overview/destinations/create/manage?dest=${data.type}`);
        }}
      />
    );
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.DESTINATIONS}
        onBackClick={handleBackPress}
      />
      <NewDestinationContainer>
        {renderSelectNewDestination()}
      </NewDestinationContainer>
    </>
  );
}
