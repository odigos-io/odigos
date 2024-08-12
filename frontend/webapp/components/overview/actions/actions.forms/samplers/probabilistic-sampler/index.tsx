import React, { useEffect } from 'react';
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
  setIsFormValid?: (value: boolean) => void;
}
const ACTION_DATA_KEY = 'actionData';

export function ProbabilisticSamplerForm({
  data,
  onChange,
  setIsFormValid = () => {},
}: ProbabilisticSamplerProps): React.JSX.Element {
  useEffect(() => {
    validateForm();
  }, [data?.sampling_percentage]);

  function handleOnChange(sampling_percentage: string): void {
    onChange(ACTION_DATA_KEY, {
      sampling_percentage,
    });
  }

  function validateForm() {
    const percentage = parseFloat(data?.sampling_percentage);
    const isValid = !isNaN(percentage) && percentage >= 0 && percentage <= 100;
    setIsFormValid(isValid);
  }

  return (
    <>
      <FormWrapper>
        <KeyvalInput
          data-cy={'create-action-sampling-percentage'}
          label="Fallback Sampling Ratio"
          value={data?.sampling_percentage}
          onChange={(value) => handleOnChange(value)}
          type="number"
          tooltip="Percentage at which items are sampled; = 100 samples all items, 0 rejects all items"
          min={0}
          max={100}
          error={
            parseFloat(data?.sampling_percentage) > 100
              ? 'Value must be less than 100'
              : ''
          }
        />
      </FormWrapper>
    </>
  );
}
