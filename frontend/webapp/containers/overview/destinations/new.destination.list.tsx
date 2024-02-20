'use client';
import React from 'react';
import { OVERVIEW, ROUTES } from '@/utils/constants';
import { OverviewHeader } from '@/components/overview';
import { DestinationSection } from '@/containers/setup/destination/destination.section';
import { NewDestinationContainer } from './destinations.styled';
import { useRouter } from 'next/navigation';

export function NewDestinationList() {
  const router = useRouter();

  return (
    <>
      <OverviewHeader
        title={OVERVIEW.MENU.DESTINATIONS}
        onBackClick={() => router.back()}
      />

      <NewDestinationContainer>
        <DestinationSection
          onSelectItem={(type) => {
            router.push(`${ROUTES.MANAGE_DESTINATION}${type}`);
          }}
        />
      </NewDestinationContainer>
    </>
  );
}
