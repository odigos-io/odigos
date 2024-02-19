'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { DestinationSection } from './destination.section';
import {
  ChooseDestinationHeader,
  SetupBackButton,
} from '@/components/setup/headers';

export function ChooseDestinationContainer() {
  const router = useRouter();

  function onDestinationSelect(type: string) {
    router.push(`/connect-destination?type=${type}`);
  }

  function onBackClick() {
    router.back();
  }

  const cardHeaderBody = () => <ChooseDestinationHeader />;

  return (
    <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
      <SetupBackButton onBackClick={onBackClick} />
      <DestinationSection onSelectItem={onDestinationSelect} />
    </KeyvalCard>
  );
}
