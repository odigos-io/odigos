'use client';
import { useEffect, useState } from 'react';
import { SelectedDestination } from '@/types';
import { ConnectionSection } from '@/containers/setup';
import { KeyvalCard, KeyvalLoader } from '@/design.system';
import { useRouter, useSearchParams } from 'next/navigation';
import { useDestinations } from '@/hooks/destinations/useDestinations';
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

  function onBackClick() {
    router.back();
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
      <SetupBackButton onBackClick={onBackClick} />
      <ConnectionSection
        supportedSignals={selectedDestination?.supported_signals}
      />
    </KeyvalCard>
  );
}
