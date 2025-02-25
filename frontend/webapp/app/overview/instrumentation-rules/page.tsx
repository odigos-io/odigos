'use client';

import React from 'react';
import { OverviewFocusedRules, OverviewHeader, OverviewModalsAndDrawers, PageContainer } from '@/components';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewFocusedRules />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
