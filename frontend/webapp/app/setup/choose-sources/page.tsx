'use client';
import React, { useState } from 'react';

import { useSuspenseQuery, gql } from '@apollo/client';
import { Dropdown, Input, SectionTitle } from '@/reuseable-components';

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

  const [selectedOption, setSelectedOption] = useState('All types');
  const options = [
    'All types',
    'Existing destinations',
    'Self hosted',
    'Managed',
  ];

  return (
    <div style={{ width: '40vw' }}>
      <SectionTitle
        title="Choose sources"
        description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations."
      />
      <div style={{ display: 'flex', gap: 8 }}>
        <Input
          placeholder="Search for sources"
          icon={'/icons/common/search.svg'}
        />

        <Dropdown
          options={options}
          selectedOption={selectedOption}
          onSelect={setSelectedOption}
          // title="Select Type"
          tooltip="Choose a type from the dropdown"
        />
      </div>
    </div>
  );
}
