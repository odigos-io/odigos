'use client';
import React from 'react';
import { OVERVIEW } from '@/utils';
import { useRouter } from 'next/navigation';
import { OverviewHeader } from '@/components';
import { ChooseActionContainer } from '@/containers';

export default function ChooseActionPage() {
  const router = useRouter();

  function onButtonClick() {
    router.back();
  }

  return (
    <>
      <OverviewHeader
        onBackClick={onButtonClick}
        title={OVERVIEW.MENU.ACTIONS}
      />
      <ChooseActionContainer />
    </>
  );
}
