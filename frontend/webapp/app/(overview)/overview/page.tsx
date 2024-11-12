'use client';
import React from 'react';
<<<<<<< HEAD
import { OverviewDataFlowContainer } from '@/containers';

=======
import dynamic from 'next/dynamic';

const OverviewDataFlowContainer = dynamic(() => import('@/containers/main/overview/overview-data-flow'), {
  ssr: false,
});

>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
export default function MainPage() {
  return (
    <>
      <OverviewDataFlowContainer />
    </>
  );
}
