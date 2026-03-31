'use client';

import React from 'react';
import { ROUTES } from '@/utils';
import { useRouter } from 'next/navigation';
import { Onboarding } from '@odigos/ui-kit/containers/v2';
import { useDestinationCRUD, useDestinationCategories, useNamespace, usePotentialDestinations, useSourceCRUD, useTestConnection } from '@/hooks';

export default function Page() {
  const router = useRouter();
  const { persistSourcesV2 } = useSourceCRUD();
  const { testConnection } = useTestConnection();
  const { getDestinationCategories } = useDestinationCategories();
  const { createDestination } = useDestinationCRUD();
  const { fetchNamespacesWithWorkloads } = useNamespace();
  const { getPotentialDestinations } = usePotentialDestinations();

  return (
    <Onboarding
      fetchNamespacesWithWorkloads={fetchNamespacesWithWorkloads}
      persistSources={persistSourcesV2}
      getDestinationCategories={getDestinationCategories}
      getPotentialDestinations={getPotentialDestinations}
      testConnection={testConnection}
      createDestination={createDestination}
      onDone={() => router.push(ROUTES.OVERVIEW)}
    />
  );
}
