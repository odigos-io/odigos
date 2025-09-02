'use client';

import React from 'react';
import { ROUTES } from '@/utils';
import { SetupSummary } from '@odigos/ui-kit/containers';
import { useDestinationCategories, useSetupHelpers } from '@/hooks';
import { OnboardingContentWrapper, SetupHeader } from '@/components';

export default function Page() {
  const { onClickRouteFromSummary } = useSetupHelpers();
  const { categories } = useDestinationCategories();

  return (
    <>
      <SetupHeader step={5} />
      <OnboardingContentWrapper>
        <SetupSummary
          categories={categories}
          onEditStream={() => onClickRouteFromSummary(ROUTES.CHOOSE_STREAM)}
          onEditSources={() => onClickRouteFromSummary(ROUTES.CHOOSE_SOURCES)}
          onEditDestinations={() => onClickRouteFromSummary(ROUTES.CHOOSE_DESTINATION)}
        />
      </OnboardingContentWrapper>
    </>
  );
}
