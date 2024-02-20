'use client';
import { useEffect, useState } from 'react';
import { useDestinations } from '@/hooks';
import { SelectedDestination } from '@/types';
import { ConnectionSection } from './connection.section';
import { KeyvalCard, KeyvalLoader } from '@/design.system';
import { useRouter, useSearchParams } from 'next/navigation';
import {
  SetupBackButton,
  ConnectDestinationHeader,
} from '@/components/setup/headers';

const SEARCH_PARAM_TYPE = 'type';

export function ConnectDestinationContainer() {
  const [selectedDestination, setSelectedDestination] =
    useState<SelectedDestination>();

  const { getCurrentDestinationByType, destinationsTypes, isLoading } =
    useDestinations();

  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(getCurrentDestination, [destinationsTypes]);

  function getCurrentDestination() {
    const type = searchParams.get(SEARCH_PARAM_TYPE);
    if (destinationsTypes) {
      const data = getCurrentDestinationByType(type as string);
      setSelectedDestination(data);
    }
  }

  if (isLoading) {
    return <KeyvalLoader />;
  }

  const cardHeaderBody = () => (
    <ConnectDestinationHeader
      icon={selectedDestination?.image_url}
      name={selectedDestination?.display_name}
    />
  );

  return (
    <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
      <SetupBackButton onBackClick={() => router.back()} />
      <ConnectionSection
        supportedSignals={selectedDestination?.supported_signals}
      />
    </KeyvalCard>
  );
}
