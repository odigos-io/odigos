import React from 'react';
import styled from 'styled-components';
import { KeyvalInput } from '@/design.system';

const FormWrapper = styled.div`
  width: 375px;
`;

interface ErrorSampler {
  fallback_sampling_ratio: number;
}

interface ErrorSamplerFormProps {
  data: ErrorSampler;
  onChange: (key: string, value: ErrorSampler | null) => void;
}
const ACTION_DATA_KEY = 'actionData';
export function ErrorSamplerForm({
  data,
  onChange,
}: ErrorSamplerFormProps): React.JSX.Element {
  function handleOnChange(fallback_sampling_ratio: number): void {
    onChange(ACTION_DATA_KEY, {
      fallback_sampling_ratio,
    });
  }

  return (
    <>
      <FormWrapper>
        <KeyvalInput
          label="Fallback Sampling Ratio"
          value={data?.fallback_sampling_ratio.toString()}
          placeholder="0"
          onChange={(value) => handleOnChange(+value)}
          type="number"
          tooltip="The percentage of errors to sample."
        />
      </FormWrapper>
    </>
  );
}
