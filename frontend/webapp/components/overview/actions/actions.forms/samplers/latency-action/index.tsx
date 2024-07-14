import React from 'react';
import styled from 'styled-components';
import { KeyvalDropDown, KeyvalInput } from '@/design.system';

const FormWrapper = styled.div`
  width: 375px;
`;

interface LatencySampler {
  fallback_sampling_ratio: number;
  minimum_latency_threshold: number;
  http_route: string;
  service_name: string;
}

interface LatencySamplerFormProps {
  data: LatencySampler;
  onChange: (key: string, value: LatencySampler | null) => void;
}
const ACTION_DATA_KEY = 'actionData';
export function LatencySamplerForm({
  data,
  onChange,
}: LatencySamplerFormProps): React.JSX.Element {
  function handleOnChange(data): void {
    onChange(ACTION_DATA_KEY, null);
  }

  return (
    <>
      <FormWrapper>
        <KeyvalDropDown
          data={[
            { id: 1, label: 'frontend' },
            { id: 2, label: 'Inventory' },
          ]}
          label="Service Name"
          value={null}
          onChange={(value) => handleOnChange(+value)}
          tooltip="Specifies the service to which the action will be applied"
        />
        <KeyvalInput
          label="Http Route"
          value={data?.http_route}
          onChange={(value) => handleOnChange(+value)}
          type="text"
          tooltip="Specifies the route to which the action will be applied"
        />
      </FormWrapper>
    </>
  );
}
