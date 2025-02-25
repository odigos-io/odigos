'use client';

import React from 'react';
import { OverviewFocusedSources, OverviewHeader, OverviewModalsAndDrawers, PageContainer } from '@/components';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewFocusedSources />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
