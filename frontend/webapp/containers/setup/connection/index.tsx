'use client';
import { useEffect, useState } from 'react';
import { ConnectionSection } from '@/containers/setup';
import { KeyvalCard, KeyvalLoader } from '@/design.system';
import { useRouter, useSearchParams } from 'next/navigation';
import { useDestinations } from '@/hooks/destinations/useDestinations';

import {
  SetupBackButton,
  ConnectDestinationHeader,
} from '@/components/setup/headers';

export function ConnectDestinationContainer() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const { getCurrentDestinationByType, destinationsTypes, isLoading } =
    useDestinations();

  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const type = searchParams.get('type');
    if (destinationsTypes) {
      const data = getCurrentDestinationByType(type as string);
      setSelectedDestination(data);
    }
  }, [destinationsTypes]);

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
      <ConnectionSection sectionData={undefined} />
    </KeyvalCard>
  );
}
