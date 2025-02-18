'use client';

import React from 'react';
import OverviewMain from '@/components/lib-imports/overview-main';
import PageContainer from '@/components/providers/page-container';
import OverviewHeader from '@/components/lib-imports/overview-header';
import OverviewModalsAndDrawers from '@/components/lib-imports/overview-modals-and-drawers';

export default function Page() {
  return (
    <PageContainer>
      <OverviewHeader />
      <OverviewMain />
      <OverviewModalsAndDrawers />
    </PageContainer>
  );
}
