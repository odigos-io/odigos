'use client';
import React, { useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { DestinationSection } from './destination.section';
import {
  SetupBackButton,
  ChooseDestinationHeader,
} from '@/components/setup/headers';

export function ChooseDestinationContainer() {
  const [totalSelectedApps, setTotalSelectedApps] = useState(0);
  const router = useRouter();

  const selectedSources = useSelector(({ app }) => app.sources);

  useEffect(calculateTotalSelectedApps, [selectedSources]);

  function calculateTotalSelectedApps() {
    let total = 0;
    for (const key in selectedSources) {
      const apps = selectedSources[key]?.objects;
      const counter = apps?.filter((item) => item.selected)?.length;
      total += counter;
    }
    setTotalSelectedApps(total);
  }

  function onDestinationSelect(type: string) {
    router.push(`/connect-destination?type=${type}`);
  }

  function onBackClick() {
    router.back();
  }

  const cardHeaderBody = () => (
    <ChooseDestinationHeader totalSelectedApps={totalSelectedApps} />
  );

  return (
    <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
      <SetupBackButton onBackClick={onBackClick} />
      <DestinationSection onSelectItem={onDestinationSelect} />
    </KeyvalCard>
  );
}
