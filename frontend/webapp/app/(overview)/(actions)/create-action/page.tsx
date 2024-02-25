'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';

export default function CreateActionPage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  console.log('object');
  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={OVERVIEW.CREATE_ACTION}
      />
    </>
  );
}
