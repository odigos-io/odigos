'use client';

import React from 'react';
import { ROUTES } from '@/utils';
import { useSetupHelpers } from '@/hooks';
import { SetupHeader } from '@/components';
import { SetupSummary } from '@odigos/ui-kit/containers';

export default function Page() {
  const { onClickRouteFromSummary } = useSetupHelpers();

  return (
    <>
      <SetupHeader step={5} />
      <SetupSummary
        onEditStream={() => onClickRouteFromSummary(ROUTES.CHOOSE_STREAM)}
        onEditSources={() => onClickRouteFromSummary(ROUTES.CHOOSE_SOURCES)}
        onEditDestinations={() => onClickRouteFromSummary(ROUTES.CHOOSE_DESTINATION)}
      />
    </>
  );
}
