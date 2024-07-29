import React, { useEffect } from 'react';
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
  setIsFormValid?: (value: boolean) => void;
}

const ACTION_DATA_KEY = 'actionData';

export function ErrorSamplerForm({
  data,
  onChange,
  setIsFormValid = () => {},
}: ErrorSamplerFormProps): React.JSX.Element {
  useEffect(() => {
    validateForm();
  }, [data?.fallback_sampling_ratio]);

  function handleOnChange(fallback_sampling_ratio: number): void {
    onChange(ACTION_DATA_KEY, {
      fallback_sampling_ratio,
    });
  }

  function validateForm() {
    const isValid =
      !isNaN(data?.fallback_sampling_ratio) &&
      data?.fallback_sampling_ratio >= 0 &&
      data?.fallback_sampling_ratio <= 100;

    setIsFormValid(isValid);
  }

  return (
    <>
      <FormWrapper>
        <KeyvalInput
          label="Fallback Sampling Ratio"
          value={data?.fallback_sampling_ratio?.toString()}
          onChange={(value) => handleOnChange(+value)}
          type="number"
          tooltip="Specifies the ratio of non-error traces you still want to retain"
          min={0}
          max={100}
          error={
            data?.fallback_sampling_ratio > 100
              ? 'Value must be less than 100'
              : ''
          }
        />
      </FormWrapper>
    </>
  );
}
