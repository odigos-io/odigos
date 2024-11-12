'use client';
import React from 'react';
import dynamic from 'next/dynamic';

const OverviewDataFlowContainer = dynamic(() => import('@/containers/main/overview/overview-data-flow'), {
  ssr: false,
});

export default function MainPage() {
  return (
    <>
      <OverviewDataFlowContainer />
    </>
  );
}
