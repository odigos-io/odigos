'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { OverviewHeader } from '@/components';
import { ManagedActionsContainer } from '@/containers';

export default function OverviewPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.ACTIONS} />
      <ManagedActionsContainer />
    </>
  );
}
