'use client';

import React from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { useSetupStore } from '@odigos/ui-kit/store';
import { DestinationSelectionForm } from '@odigos/ui-kit/containers';
import { useDestinationCategories, useDestinationCRUD, usePotentialDestinations, useTestConnection } from '@/hooks';

export default function Page() {
  const router = useRouter();
  const { configuredSources } = useSetupStore();

  const { testConnection } = useTestConnection();
  const { categories } = useDestinationCategories();
  const { updateDestination } = useDestinationCRUD();
  const { potentialDestinations } = usePotentialDestinations();

  return (
    <>
      <SetupHeader step={4} />
      <DestinationSelectionForm
        categories={categories}
        potentialDestinations={potentialDestinations}
        updateDestination={updateDestination}
        testConnection={testConnection}
        isSourcesListEmpty={!Object.values(configuredSources).some((sources) => sources.length)}
        goToSources={() => router.push(ROUTES.CHOOSE_SOURCES)}
      />
    </>
  );
}
