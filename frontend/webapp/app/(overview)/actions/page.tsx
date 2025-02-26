'use client';

import React from 'react';
import { OverviewFocusedActions, OverviewHeader, OverviewModalsAndDrawers, PageContainer } from '@/components';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewFocusedActions />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
