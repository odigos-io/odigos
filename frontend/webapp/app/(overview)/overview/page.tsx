'use client';

import React from 'react';
import { OverviewHeader, OverviewMain, OverviewModalsAndDrawers, PageContainer } from '@/components';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewMain />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
