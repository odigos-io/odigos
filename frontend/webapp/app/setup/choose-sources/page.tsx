'use client';
import React, { useEffect, useState } from 'react';

import { useSuspenseQuery, gql } from '@apollo/client';
import {
  Checkbox,
  Counter,
  Divider,
  Dropdown,
  Input,
  SectionTitle,
  Toggle,
} from '@/reuseable-components';

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
    console.log(data);
  }, [data]);

  const [selectedOption, setSelectedOption] = useState('All types');
  const options = [
    'All types',
    'Existing destinations',
    'Self hosted',
    'Managed',
  ];

  const handleCheckboxChange = (value: boolean) => {
    console.log('Checkbox is now', value);
  };

  return (
    <div style={{ width: 756 }}>
      <SectionTitle
        title="Choose sources"
        description="Apps will be automatically instrumented, and data will be sent to the relevant APM's destinations."
      />
      <div style={{ display: 'flex', gap: 8, marginTop: 24 }}>
        <Input
          placeholder="Search for sources"
          icon={'/icons/common/search.svg'}
        />

        <Dropdown
          options={options}
          selectedOption={selectedOption}
          onSelect={setSelectedOption}
        />
      </div>
      <Divider thickness={1} margin="24px 0 16px" />
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
        <Counter value={0} title="Selected apps" />
        <div style={{ display: 'flex', gap: 8 }}>
          <Toggle title="Auto-instrument all apps" />
          <Toggle title="Show selected only" />
        </div>
        <Checkbox
          title="Future apps"
          tooltip="Automatically instrument all future apps"
          initialValue={false}
          onChange={handleCheckboxChange}
        />
      </div>
      <Divider thickness={1} margin="24px 0 16px" />
    </div>
  );
}
