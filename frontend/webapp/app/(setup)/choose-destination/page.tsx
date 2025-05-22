'use client';

import React, { useMemo } from 'react';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { useSetupStore } from '@odigos/ui-kit/store';
import { DestinationSelectionForm } from '@odigos/ui-kit/containers';
import { useDestinationCategories, usePotentialDestinations, useSetupHelpers, useTestConnection } from '@/hooks';

export default function Page() {
  const { configuredSources } = useSetupStore();
  const { testConnection } = useTestConnection();
  const { categories } = useDestinationCategories();
  const { potentialDestinations } = usePotentialDestinations();
  const { onClickSummary, onClickRouteFromSummary } = useSetupHelpers();

  const isSourcesListEmpty = useMemo(() => !Object.values(configuredSources).some((sources) => sources.length), [configuredSources]);
  const goToSources = () => onClickRouteFromSummary(ROUTES.CHOOSE_SOURCES);

  return (
    <>
      <SetupHeader step={4} />
      <DestinationSelectionForm
        categories={categories}
        potentialDestinations={potentialDestinations}
        testConnection={testConnection}
        isSourcesListEmpty={isSourcesListEmpty}
        goToSources={goToSources}
        onClickSummary={onClickSummary}
      />
    </>
  );
}
