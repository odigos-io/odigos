'use client';
import React, { useEffect } from 'react';
import { ROUTES } from '@/utils';
import { setSources } from '@/store';
import { useSectionData } from '@/hooks';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { SourcesSection } from './sources.section';
import { useDispatch, useSelector } from 'react-redux';
import { ChooseSourcesHeader } from '@/components/setup/headers';
export function ChooseSourcesContainer() {
  const router = useRouter();

  const dispatch = useDispatch();
  const selectedSources = useSelector(({ app }) => app.sources);

  const { sectionData, setSectionData, totalSelected } = useSectionData({});

  useEffect(onload, []);

  function onload() {
    if (selectedSources) {
      setSectionData({ ...selectedSources });
    }
    console.log({ selectedSources });
  }

  async function onNextClick() {
    dispatch(setSources(sectionData));
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
