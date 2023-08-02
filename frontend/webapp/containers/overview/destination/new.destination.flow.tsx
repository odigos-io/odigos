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

export function NewDestinationFlow({ onSuccess, onError }) {
  const { sectionData, setSectionData } = useSectionData(null);

  const { mutate } = useMutation((body) => setDestination(body));
  const router = useRouter();

  // function onSubmit(newDestination) {
  //   const destination = {
  //     ...newDestination,
  //     type: sectionData.type,
  //   };

  //   mutate(destination, {
  //     onSuccess: () => onSuccess(OVERVIEW.DESTINATION_CREATED_SUCCESS),
  //     onError,
  //   });
  // }

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
