'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';
import { EditInstrumentationRuleContainer } from '@/containers';

export default function EditInstrumentationRulePage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={'Edit Instrumentation Rule'}
      />
      <EditInstrumentationRuleContainer />
    </>
  );
}
