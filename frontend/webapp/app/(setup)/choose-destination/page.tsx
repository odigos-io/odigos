'use client';

import React, { useMemo } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { SetupHeader } from '@/components';
import { useSetupStore } from '@odigos/ui-kit/store';
import { ROUTES, SKIP_TO_SUMMERY_QUERY_PARAM } from '@/utils';
import { DestinationSelectionForm } from '@odigos/ui-kit/containers';
import { useDestinationCategories, usePotentialDestinations, useTestConnection } from '@/hooks';

export default function Page() {
  const router = useRouter();
  const params = useSearchParams();
  const skipToSummary = !!params.get(SKIP_TO_SUMMERY_QUERY_PARAM);

  const { configuredSources } = useSetupStore();
  const { testConnection } = useTestConnection();
  const { categories } = useDestinationCategories();
  const { potentialDestinations } = usePotentialDestinations();

  const isSourcesListEmpty = useMemo(() => {
    return !Object.values(configuredSources).some((sources) => sources.length);
  }, [configuredSources]);

  const goToSources = () => {
    const querystring = skipToSummary ? `?${SKIP_TO_SUMMERY_QUERY_PARAM}=true` : '';
    router.push(ROUTES.CHOOSE_SOURCES + querystring);
  };

  // If we do not want to show the "go to summary button", we have to pass undefined as prop
  const onClickSummary = skipToSummary
    ? () => {
        router.push(ROUTES.SETUP_SUMMARY);
      }
    : undefined;

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
