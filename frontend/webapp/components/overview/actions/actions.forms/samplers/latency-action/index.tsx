import React from 'react';
import styled from 'styled-components';
import { KeyvalDropDown, KeyvalInput } from '@/design.system';
import { useSources } from '@/hooks';

const FormWrapper = styled.div`
  width: 375px;
`;

const InputsWrapper = styled.div`
  display: flex;
  flex-direction: column;
  gap: 15px;
  justify-content: space-between;
  margin-bottom: 15px;
`;

const DropdownWrapper = styled.div`
  margin-bottom: 15px;
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
  const { sources } = useSources();

  const memoizedSources = React.useMemo(() => {
    return sources.map((source, index) => ({
      id: index,
      label: source.name,
    }));
  }, [sources]);

  function handleOnChange(key, value): void {
    const newData = {
      ...data,
      [key]: value,
    };
    console.log({ newData });
    onChange(ACTION_DATA_KEY, newData);
  }

  return (
    <>
      <FormWrapper>
        <DropdownWrapper>
          <KeyvalDropDown
            data={memoizedSources}
            label="Service Name"
            onChange={(value) => handleOnChange('service_name', value.label)}
          />
        </DropdownWrapper>
        <InputsWrapper>
          <KeyvalInput
            label="Http Route"
            value={data?.http_route}
            onChange={(value) => handleOnChange('http_route', value)}
            type="text"
          />
          <KeyvalInput
            label="Minimum Latency Threshold"
            value={data?.minimum_latency_threshold?.toString()}
            onChange={(value) =>
              handleOnChange('minimum_latency_threshold', +value)
            }
            type="number"
            min={0}
            error={data?.minimum_latency_threshold < 0 ? 'Invalid value' : ''}
          />
          <KeyvalInput
            label="Fallback Sampling Ratio"
            value={data?.fallback_sampling_ratio?.toString()}
            onChange={(value) =>
              handleOnChange('fallback_sampling_ratio', +value)
            }
            min={0}
            max={100}
            type="number"
            error={
              data?.fallback_sampling_ratio > 100
                ? 'Value must be less than 100'
                : ''
            }
          />
        </InputsWrapper>
      </FormWrapper>
    </>
  );
}
