"use client";
import React, { useEffect } from "react";
import { NOTIFICATION, OVERVIEW, QUERIES } from "@/utils/constants";
import { useMutation, useQuery } from "react-query";
import {
  getDestination,
  getDestinationsTypes,
  setDestination,
} from "@/services";
import { ManageDestination, OverviewHeader } from "@/components/overview";
import { useNotification, useSectionData } from "@/hooks";
import { useRouter, useSearchParams } from "next/navigation";
import { styled } from "styled-components";

const DEST = "dest";

const NewDestinationContainer = styled.div`
  padding: 20px 36px;
`;

export default function NewDestinationFlow() {
  const { sectionData, setSectionData } = useSectionData(null);
  const searchParams = useSearchParams();
  const { data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, sectionData?.type],
    () => getDestination(sectionData?.type),
    {
      enabled: !!sectionData,
    }
  );

  const { isLoading, data } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );
  const { show, Notification } = useNotification();
  const { mutate } = useMutation((body) => setDestination(body));
  const router = useRouter();

  useEffect(onPageLoad, [data]);

  function onPageLoad() {
    const search = searchParams.get(DEST);
    let currentData = null;
    data?.categories.forEach((item) => {
      const filterItem = item.items.filter((dest) => dest?.type === search);
      if (filterItem.length) {
        currentData = filterItem[0];
      }
    });
    setSectionData(currentData);
  }

  function onSuccess(message = OVERVIEW.DESTINATION_UPDATE_SUCCESS) {
    show({
      type: NOTIFICATION.SUCCESS,
      message,
    });
  }

  function onError({ response }) {
    const message = response?.data?.message;
    show({
      type: NOTIFICATION.ERROR,
      message,
    });
  }

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
    router.back();
  }

  if (isLoading) {
    return;
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.DESTINATIONS}
        onBackClick={handleBackPress}
      />
      {destinationType && sectionData && (
        <NewDestinationContainer>
          <ManageDestination
            destinationType={destinationType}
            selectedDestination={sectionData}
            onSubmit={onSubmit}
          />
        </NewDestinationContainer>
      )}
      <Notification />
    </>
  );
}
