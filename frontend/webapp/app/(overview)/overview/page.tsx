'use client';
import React from 'react';
import dynamic from 'next/dynamic';

const ToastList = dynamic(() => import('@/components/notification/toast-list'), { ssr: false });
const AllDrawers = dynamic(() => import('@/components/overview/all-drawers'), { ssr: false });
const AllModals = dynamic(() => import('@/components/overview/all-modals'), { ssr: false });
const OverviewDataFlowContainer = dynamic(() => import('@/containers/main/overview/overview-data-flow'), { ssr: false });

export default function MainPage() {
  return (
    <>
      <ToastList />
      <AllDrawers />
      <AllModals />
      <OverviewDataFlowContainer />
    </>
  );
}
