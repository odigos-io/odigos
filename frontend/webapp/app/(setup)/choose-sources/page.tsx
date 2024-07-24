'use client';
import React, { useEffect } from 'react';
import { StepsList } from '@/components';
import { ChooseSourcesContainer } from '@/containers';
import { CardWrapper, PageContainer, StepListWrapper } from '../styled';

import { useSuspenseQuery, gql } from '@apollo/client';

const GET_COMPUTE_PLATFORM = gql`
  query GetComputePlatform($cpId: ID!) {
    computePlatform(cpId: $cpId) {
      id
      name
      computePlatformType
      k8sActualSources {
        namespace
        kind
        name
        serviceName
        autoInstrumented
        creationTimestamp
        numberOfInstances
        hasInstrumentedApplication
        instrumentedApplicationDetails {
          languages {
            containerName
            language
          }
          conditions {
            type
            status
            lastTransitionTime
            reason
            message
          }
        }
      }
    }
  }
`;

export default function ChooseSourcesPage() {
  const { error, data } = useSuspenseQuery(GET_COMPUTE_PLATFORM, {
    variables: { cpId: '1' },
  });

  useEffect(() => {
    if (error) {
      console.error(error);
    }
    console.log({ data });
  }, [error, data]);
  return (
    <PageContainer>
      <StepListWrapper>
        <StepsList currentStepIndex={0} />
      </StepListWrapper>
      <CardWrapper>
        <ChooseSourcesContainer />
      </CardWrapper>
    </PageContainer>
  );
}
