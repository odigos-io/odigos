'use client';
import React from 'react';
import { useRouter } from 'next/navigation';
import { KeyvalCard } from '@/design.system';
import { StepsList } from '@/components/lists';
import { NOTIFICATION, SETUP } from '@/utils/constants';
import { ChooseSourcesHeader } from '@/components/setup/headers';
import { useNotification, useSectionData, useSources } from '@/hooks';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';
import { SourcesSection } from '@/containers/setup/sources/sources.section';

export default function ChooseSourcesPage() {
  const router = useRouter();
  const { upsertSources } = useSources();
  const { show, Notification } = useNotification();
  const { sectionData, setSectionData, totalSelected } = useSectionData({});
  async function onNextClick() {
    upsertSources({
      sectionData,
      onSuccess: () => router.push('/choose-destination'),
      onError: ({ response }) => {
        const message = response?.data?.message || SETUP.ERROR;
        show({
          type: NOTIFICATION.ERROR,
          message,
        });
      },
    });
  }

  const cardHeaderBody = () => (
    <ChooseSourcesHeader
      onNextClick={onNextClick}
      totalSelected={totalSelected}
    />
  );

  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={0} />
      </StepListWrapper>
      <CardWrapper>
        <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
          <SourcesSection
            sectionData={sectionData}
            setSectionData={setSectionData}
          />
        </KeyvalCard>
      </CardWrapper>
      <Notification />
    </PageContainer>
  );
}
