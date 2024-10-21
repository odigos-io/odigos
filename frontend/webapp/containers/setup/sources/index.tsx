'use client';
import React, { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { useAppStore } from '@/store';
import { useSectionData } from '@/hooks';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { SourcesSection } from './sources.section';
import { ChooseSourcesHeader } from '@/components/setup/headers';
export function ChooseSourcesContainer() {
  const router = useRouter();

  const selectedSources = useAppStore((state) => state.sources);
  const setSources = useAppStore((state) => state.setSources);

  const { sectionData, setSectionData, totalSelected } = useSectionData({});

  useEffect(onload, []);

  function onload() {
    selectedSources && setSectionData({ ...selectedSources });
  }

  async function onNextClick() {
    setSources(sectionData);
    router.push(ROUTES.CHOOSE_DESTINATION);
  }

  const cardHeaderBody = () => (
    <ChooseSourcesHeader
      onNextClick={onNextClick}
      totalSelected={totalSelected}
    />
  );

  return (
    <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
      <SourcesSection
        sectionData={sectionData}
        setSectionData={setSectionData}
      />
    </KeyvalCard>
  );
}
