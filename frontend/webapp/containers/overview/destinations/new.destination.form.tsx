'use client';
import React, { useEffect } from 'react';
import { styled } from 'styled-components';
import { HideScroll } from '@/styles/styled';
import { useMutation, useQuery } from 'react-query';
import { useNotification, useSectionData } from '@/hooks';
import { useRouter, useSearchParams } from 'next/navigation';
import { ManageDestination, OverviewHeader } from '@/components/overview';
import { NOTIFICATION, OVERVIEW, QUERIES, ROUTES } from '@/utils/constants';
import {
  getDestination,
  getDestinationsTypes,
  setDestination,
} from '@/services';

const DEST = 'dest';

const NewDestinationContainer = styled.div`
  padding: 20px 36px;

  overflow: scroll;
  ::-webkit-scrollbar {
    display: none;
  }
  -ms-overflow-style: none;
  scrollbar-width: none;
  height: 80vh;
  @media screen and (max-height: 750px) {
    height: 85vh;
  }
`;

export function NewDestinationForm() {
  const { sectionData, setSectionData } = useSectionData(null);
  const { show, Notification } = useNotification();
  const { mutate } = useMutation((body) => setDestination(body));
  const searchParams = useSearchParams();
  const router = useRouter();

  const { data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, sectionData?.type],
    () => getDestination(sectionData?.type),
    {
      enabled: !!sectionData,
    }
  );

  const { data: destinationsList } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  useEffect(onPageLoad, [destinationsList]);

  function onPageLoad() {
    const search = searchParams.get(DEST);
    if (!destinationsList || !search) return;

    let currentData = null;

    for (const category of destinationsList.categories) {
      if (currentData) {
        break;
      }
      const filterItem = category.items.filter(({ type }) => type === search);
      if (filterItem.length) {
        currentData = filterItem[0];
      }
    }

    setSectionData(currentData);
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

    console.log({ newDestination });

    // return;
    mutate(destination, {
      onSuccess: () => router.push(`${ROUTES.DESTINATIONS}?status=created`),
      onError,
    });
  }

  function handleBackPress() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.DESTINATIONS}
        onBackClick={handleBackPress}
      />
      <HideScroll>
        {destinationType && sectionData && (
          <NewDestinationContainer>
            <ManageDestination
              destinationType={destinationType}
              selectedDestination={sectionData}
              onSubmit={onSubmit}
            />
          </NewDestinationContainer>
        )}
      </HideScroll>
      <Notification />
    </>
  );
}
