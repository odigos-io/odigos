"use client";
import React, { useState } from "react";
import { OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, setDestination } from "@/services";
import { ManageDestination, OverviewHeader } from "@/components/overview";
import { useSectionData } from "@/hooks";
import { DestinationSection } from "@/containers/setup/destination/destination.section";
import { NewDestinationContainer } from "./destination.styled";

export function NewDestinationFlow({ onBackClick, onSuccess, onError }) {
  const { sectionData, setSectionData } = useSectionData(null);
  const [managed, setManaged] = useState<any>(null);
  const { data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, sectionData?.type],
    () => getDestination(sectionData?.type),
    {
      enabled: !!sectionData,
    }
  );
  const { mutate } = useMutation((body) => setDestination(body));

  function onSubmit(newDestination) {
    const destination = {
      ...newDestination,
      type: sectionData.type,
    };

    mutate(destination, {
      onSuccess: () => onSuccess(OVERVIEW.DESTINATION_CREATED_SUCCESS),
      onError,
    });
  }

  function handleBackPress() {
    if (managed && sectionData) {
      setManaged(false);
      setSectionData(null);
      return;
    }
    onBackClick();
  }

  function renderNewDestinationForm() {
    return (
      <ManageDestination
        destinationType={destinationType}
        selectedDestination={sectionData}
        onSubmit={onSubmit}
      />
    );
  }

  function renderSelectNewDestination() {
    return (
      <>
        <DestinationSection
          sectionData={sectionData}
          setSectionData={(data) => {
            setSectionData(data);
            setManaged(true);
          }}
        />
      </>
    );
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.DESTINATIONS}
        onBackClick={handleBackPress}
      />
      <NewDestinationContainer>
        {managed && sectionData
          ? renderNewDestinationForm()
          : renderSelectNewDestination()}
      </NewDestinationContainer>
    </>
  );
}
