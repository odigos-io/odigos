'use client';

import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { ENTITY_TYPES } from '@odigos/ui-utils';
import { DestinationSelectionForm, useSetupStore } from '@odigos/ui-containers';
import { useDestinationCategories, useDestinationCRUD, usePotentialDestinations, useTestConnection } from '@/hooks';

export default function Page() {
  const router = useRouter();
  const { configuredSources } = useSetupStore();

  const { categories } = useDestinationCategories();
  const { createDestination } = useDestinationCRUD();
  const { potentialDestinations } = usePotentialDestinations();
  const { testConnection, loading: testLoading, data: testResult } = useTestConnection();

  // we need this state, because "loading" from CRUD hooks is a bit delayed, and allows the user to double-click, as well as see elements render in the UI when they should not be rendered.
  const [isLoading, setIsLoading] = useState(false);

  return (
    <>
      <SetupHeader entityType={ENTITY_TYPES.DESTINATION} isLoading={isLoading} setIsLoading={setIsLoading} />
      <DestinationSelectionForm
        categories={categories}
        potentialDestinations={potentialDestinations}
        createDestination={createDestination}
        isLoading={isLoading}
        testConnection={testConnection}
        testLoading={testLoading}
        testResult={testResult}
        isSourcesListEmpty={!Object.values(configuredSources).some((sources) => !!sources.length)}
        goToSources={() => router.push(ROUTES.CHOOSE_SOURCES)}
      />
    </>
  );
}
