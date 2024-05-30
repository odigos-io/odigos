'use client';
import React, { useEffect, useMemo, useState } from 'react';
import { QUERIES, ROUTES } from '@/utils';
import { HideScroll } from '@/styles/styled';
import { KeyvalLoader } from '@/design.system';
import { ManageDestination } from '@/components';
import { useMutation, useQuery } from 'react-query';
import { useRouter, useSearchParams } from 'next/navigation';
import { getDestination, updateDestination } from '@/services';
import { ManageDestinationWrapper } from './destinations.styled';
import { deleteDestination, getDestinations } from '@/services/destinations';
const DEST = 'dest';

export function UpdateDestinationFlow() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);

  const router = useRouter();
  const searchParams = useSearchParams();

  const manageData = useMemo(() => {
    return {
      ...selectedDestination,
      ...selectedDestination?.destination_type,
    };
  }, [selectedDestination]);

  const { isLoading: destinationTypeLoading, data: destinationType } = useQuery(
    [QUERIES.API_DESTINATION_TYPE, selectedDestination?.type],
    () => getDestination(selectedDestination?.type),
    {
      enabled: !!selectedDestination,
    }
  );

  const { isLoading: destinationLoading, data: destinationList } = useQuery(
    [QUERIES.API_DESTINATIONS],
    getDestinations
  );

  const { mutate: handleUpdateDestination } = useMutation((body) =>
    updateDestination(body, selectedDestination?.id)
  );

  const { mutate: handleDeleteDestination } = useMutation((body) =>
    deleteDestination(selectedDestination?.id)
  );

  useEffect(onPageLoad, [searchParams, destinationList]);

  function onDelete() {
    handleDeleteDestination(selectedDestination.id, {
      onSuccess: () => router.push(`${ROUTES.DESTINATIONS}?status=deleted`),
    });
  }

  function onSubmit(updatedDestination) {
    const newDestinations = {
      ...updatedDestination,
      type: selectedDestination.type,
    };

    handleUpdateDestination(newDestinations, {
      onSuccess: () => router.push(`${ROUTES.DESTINATIONS}?status=updated`),
    });
  }

  function onPageLoad() {
    const search = searchParams.get(DEST);
    const currentDestination = destinationList?.filter(
      ({ id }) => id === search
    );
    if (currentDestination?.length) {
      setSelectedDestination(currentDestination[0]);
    }
  }

  if (destinationLoading || !selectedDestination) {
    return <KeyvalLoader />;
  }

  return destinationTypeLoading ? (
    <KeyvalLoader />
  ) : (
    <HideScroll>
      <ManageDestinationWrapper>
        <ManageDestination
          onBackClick={() => router.back()}
          destinationType={destinationType}
          selectedDestination={manageData}
          onSubmit={onSubmit}
          onDelete={onDelete}
        />
      </ManageDestinationWrapper>
    </HideScroll>
  );
}
