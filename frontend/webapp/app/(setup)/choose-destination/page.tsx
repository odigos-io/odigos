'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { EntityTypes } from '@odigos/ui-kit/types';
import { useSetupStore } from '@odigos/ui-kit/store';
import { DestinationSelectionForm } from '@odigos/ui-kit/containers';
import { useDestinationCategories, useDestinationCRUD, usePotentialDestinations, useTestConnection } from '@/hooks';

export default function Page() {
  const router = useRouter();
  const { configuredSources } = useSetupStore();

  const { categories } = useDestinationCategories();
  const { createDestination } = useDestinationCRUD();
  const { potentialDestinations } = usePotentialDestinations();
  const { testConnection, testConnectionResult, isTestConnectionLoading } = useTestConnection();

  // we need this state, because "loading" from CRUD hooks is a bit delayed, and allows the user to double-click, as well as see elements render in the UI when they should not be rendered.
  const [isLoading, setIsLoading] = useState(false);

  return (
    <>
      <SetupHeader entityType={EntityTypes.Destination} isLoading={isLoading} setIsLoading={setIsLoading} />
      <DestinationSelectionForm
        categories={categories}
        potentialDestinations={potentialDestinations}
        createDestination={createDestination}
        isLoading={isLoading}
        testConnection={testConnection}
        testResult={testConnectionResult}
        testLoading={isTestConnectionLoading}
        isSourcesListEmpty={!Object.values(configuredSources).some((sources) => sources.length)}
        goToSources={() => router.push(ROUTES.CHOOSE_SOURCES)}
      />
    </>
  );
}
