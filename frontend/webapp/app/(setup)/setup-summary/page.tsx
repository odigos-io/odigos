'use client';

import React from 'react';
import { ROUTES } from '@/utils';
import { SetupHeader } from '@/components';
import { SetupSummary } from '@odigos/ui-kit/containers';
import { useDestinationCategories, useSetupHelpers } from '@/hooks';

export default function Page() {
  const { onClickRouteFromSummary } = useSetupHelpers();
  const { categories } = useDestinationCategories();

  return (
    <>
      <SetupHeader step={5} />
      <SetupSummary
        categories={categories}
        onEditStream={() => onClickRouteFromSummary(ROUTES.CHOOSE_STREAM)}
        onEditSources={() => onClickRouteFromSummary(ROUTES.CHOOSE_SOURCES)}
        onEditDestinations={() => onClickRouteFromSummary(ROUTES.CHOOSE_DESTINATION)}
      />
    </>
  );
}
