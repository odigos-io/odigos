'use client';

import React, { type PropsWithChildren } from 'react';
import styled from 'styled-components';
import { useSSE, useTokenTracker } from '@/hooks';
import { ToastList } from '@odigos/ui-containers';
import { FlexColumn } from '@odigos/ui-components';
import { ErrorBoundary, OverviewHeader, OverviewModalsAndDrawers } from '@/components';

const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

function OverviewLayout({ children }: PropsWithChildren) {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  return (
    <ErrorBoundary>
      <PageContent>
        <OverviewHeader />

        {children}

        <OverviewModalsAndDrawers />
        <ToastList />
      </PageContent>
    </ErrorBoundary>
  );
}

export default OverviewLayout;
