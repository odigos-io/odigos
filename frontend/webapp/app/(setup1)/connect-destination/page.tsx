'use client';
import { useEffect, useState } from 'react';
import { StepListWrapper } from '../styled';
import { KeyvalCard, KeyvalLoader } from '@/design.system';
import { StepsList } from '@/components/lists';
import { ConnectionSection } from '@/containers/setup';
import { ConnectDestinationHeader } from '@/components/setup/headers';
import { useDestinations } from '@/hooks/destinations/useDestinations';
import { useSearchParams } from 'next/navigation';

export default function ConnectDestinationPage() {
  const [selectedDestination, setSelectedDestination] = useState<any>(null);
  const { getCurrentDestinationByType, destinationsTypes, isLoading } =
    useDestinations();

  const searchParams = useSearchParams();

  useEffect(() => {
    const type = searchParams.get('type');
    if (destinationsTypes) {
      const data = getCurrentDestinationByType(type as string);
      setSelectedDestination(data);
    }
  }, [destinationsTypes]);

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
    <>
      <StepListWrapper>
        <StepsList currentStepIndex={2} />
      </StepListWrapper>
      <KeyvalCard type={'secondary'} header={{ body: cardHeaderBody }}>
        <div style={{ padding: '0 40px', minWidth: '70vw', maxHeight: '80vh' }}>
          <ConnectionSection sectionData={undefined} />
        </div>
      </KeyvalCard>
    </>
  );
}
