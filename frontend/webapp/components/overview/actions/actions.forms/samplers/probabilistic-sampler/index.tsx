import React from 'react';
import styled from 'styled-components';
import { KeyvalInput } from '@/design.system';

const FormWrapper = styled.div`
  width: 375px;
`;

interface ProbabilisticSampler {
  sampling_percentage: string;
}

interface ProbabilisticSamplerProps {
  data: ProbabilisticSampler;
  onChange: (key: string, value: ProbabilisticSampler | null) => void;
}
const ACTION_DATA_KEY = 'actionData';
export function ProbabilisticSamplerForm({
  data,
  onChange,
}: ProbabilisticSamplerProps): React.JSX.Element {
  console.log({ data });

  function handleOnChange(sampling_percentage: string): void {
    onChange(ACTION_DATA_KEY, {
      sampling_percentage,
    });
  }

  return (
    <>
      <FormWrapper>
        <KeyvalInput
          label="Fallback Sampling Ratio"
          value={data?.sampling_percentage}
          onChange={(value) => handleOnChange(value)}
          type="number"
          tooltip="Percentage at which items are sampled; = 100 samples all items, 0 rejects all items"
        />
      </FormWrapper>
    </>
  );
}
