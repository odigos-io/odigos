'use client';

import React from 'react';
import { OverviewFocusedDestinations, OverviewHeader, OverviewModalsAndDrawers, PageContainer } from '@/components';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewFocusedDestinations />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
