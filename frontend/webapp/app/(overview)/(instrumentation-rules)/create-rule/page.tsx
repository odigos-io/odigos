'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';
import { CreateInstrumentationRulesContainer } from '@/containers';

export default function CreateActionPage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={OVERVIEW.CREATE_INSTRUMENTATION_RULE}
      />
      <CreateInstrumentationRulesContainer />
    </>
  );
}
