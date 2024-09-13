'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { OverviewHeader } from '@/components';
import { ManagedInstrumentationRulesContainer } from '@/containers/main/instrumentationrules';

export default function InstrumentationRulesPage() {
  return (
    <>
      <OverviewHeader title={OVERVIEW.MENU.INSTRUMENTATION_RULES} />
      <ManagedInstrumentationRulesContainer />
    </>
  );
}
