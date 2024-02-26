'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';
import { CreateActionContainer } from '@/containers';

export default function CreateActionPage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={OVERVIEW.CREATE_ACTION}
      />
      <CreateActionContainer />
    </>
  );
}
