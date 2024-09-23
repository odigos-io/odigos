'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';
import { ChooseInstrumentationRuleContainer } from '@/containers';

export default function ChooseInstrumentationRulesPage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={OVERVIEW.MENU.INSTRUMENTATION_RULES}
      />
      <ChooseInstrumentationRuleContainer />
    </>
  );
}
