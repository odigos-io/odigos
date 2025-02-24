import React, { type FC, type PropsWithChildren } from 'react';
import styled from 'styled-components';
import ErrorBoundary from './error-boundary';
import { useSSE, useTokenTracker } from '@/hooks';
import { ToastList } from '@odigos/ui-containers';
import { FlexColumn } from '@odigos/ui-components';

export const PageContent = styled(FlexColumn)`
  width: 100%;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.primary};
  align-items: center;
`;

const PageContainer: FC<PropsWithChildren> = ({ children }) => {
  // call important hooks that should run on page-mount
  useSSE();
  useTokenTracker();

  return (
    <ErrorBoundary>
      <PageContent>
        <ToastList />
        {children}
      </PageContent>
    </ErrorBoundary>
  );
};

export default PageContainer;
