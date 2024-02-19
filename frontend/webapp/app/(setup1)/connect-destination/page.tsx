'use client';
import { useEffect, useState } from 'react';
import { StepsList } from '@/components/lists';
import { ConnectionSection } from '@/containers/setup';
import { KeyvalCard, KeyvalLoader } from '@/design.system';
import { useRouter, useSearchParams } from 'next/navigation';
import { useDestinations } from '@/hooks/destinations/useDestinations';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';
import {
  SetupBackButton,
  ConnectDestinationHeader,
} from '@/components/setup/headers';

export default function ConnectDestinationPage() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const { getCurrentDestinationByType, destinationsTypes, isLoading } =
    useDestinations();

  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const type = searchParams.get('type');
    if (destinationsTypes) {
      const data = getCurrentDestinationByType(type as string);
      setSelectedDestination(data);
    }
  }, [destinationsTypes]);

  function onBackClick() {
    router.back();
  }

  if (isLoading) {
    return <KeyvalLoader />;
  }

  const cardHeaderBody = () => (
    <ConnectDestinationHeader
      icon={selectedDestination?.image_url}
      name={selectedDestination?.display_name}
    />
  );

  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={2} />
      </StepListWrapper>
      <CardWrapper>
        <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
          <SetupBackButton onBackClick={onBackClick} />
          <ConnectionSection sectionData={undefined} />
        </KeyvalCard>
      </CardWrapper>
    </PageContainer>
  );
}
