"use client";
import React, { useState } from "react";
import { KeyvalLoader, KeyvalText } from "@/design.system";
import { NOTIFICATION, OVERVIEW, QUERIES, SETUP } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import { getDestination, setDestination, updateDestination } from "@/services";
import { ManageDestination, OverviewHeader } from "@/components/overview";
import { useNotification, useSectionData } from "@/hooks";
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
export function NewDestinationFlow({ onBackClick }) {
  const { sectionData, setSectionData } = useSectionData(null);
  const [managed, setManaged] = useState<any>(null);
  const { show, Notification } = useNotification();

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

    function onSuccess() {
      onBackClick();
    }

    function onError({ response }) {
      const message = response?.data?.message;
      show({
        type: NOTIFICATION.ERROR,
        message,
      });
    }

    mutate(destination, {
      onSuccess,
      onError,
    });
  }

  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.DESTINATIONS} />
      <NewDestinationContainer>
        {managed && sectionData ? (
          <ManageDestination
            destinationType={destinationType}
            selectedDestination={sectionData}
            onSubmit={onSubmit}
            onBackClick={() => {
              setManaged(false);
              setSectionData(null);
            }}
          />
        ) : (
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
        )}
        <Notification />
      </NewDestinationContainer>
    </>
  );
}
