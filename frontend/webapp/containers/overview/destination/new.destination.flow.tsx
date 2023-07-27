"use client";
import React, { useState } from "react";
import { KeyvalText } from "@/design.system";
import { OVERVIEW, QUERIES, SETUP } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, setDestination } from "@/services";
import { ManageDestination, OverviewHeader } from "@/components/overview";
import { useSectionData } from "@/hooks";
import { DestinationSection } from "@/containers/setup/destination/destination.section";
import { NewDestinationContainer } from "./destination.styled";
import { Back } from "@/assets/icons/overview";
import { styled } from "styled-components";

const BackButtonWrapper = styled.div`
  display: flex;
  align-items: center;
  cursor: pointer;
  p {
    cursor: pointer !important;
  }
`;
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
      onSuccess,
      onError,
    });
  }

  function renderNewDestinationForm() {
    return (
      <ManageDestination
        destinationType={destinationType}
        selectedDestination={sectionData}
        onSubmit={onSubmit}
        onBackClick={() => {
          setManaged(false);
          setSectionData(null);
        }}
      />
    );
  }

  function renderSelectNewDestination() {
    return (
      <>
        <BackButtonWrapper onClick={onBackClick}>
          <Back width={14} />
          <KeyvalText size={14}>{SETUP.BACK}</KeyvalText>
        </BackButtonWrapper>
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
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <NewDestinationContainer>
        {managed && sectionData
          ? renderNewDestinationForm()
          : renderSelectNewDestination()}
      </NewDestinationContainer>
    </>
  );
}
