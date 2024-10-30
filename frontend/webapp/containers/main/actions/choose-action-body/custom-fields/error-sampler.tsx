import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { Input } from '@/reuseable-components';
import type { ErrorSamplerSpec } from '@/types';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = ErrorSamplerSpec;

const MIN = 0,
  MAX = 100;

const ErrorSampler: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { fallback_sampling_ratio: 0 }).fallback_sampling_ratio, [value]);

  const handleChange = (val: string) => {
    let num = Number(val);

    if (Number.isNaN(num) || num < MIN || num > MAX) {
      num = MIN;
    }

    const payload: Parsed = {
      fallback_sampling_ratio: num,
    };

    const str = !!payload.fallback_sampling_ratio ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return (
    <Input
      title='Fallback sampling ratio'
      required
      type='number'
      min={MIN}
      max={MAX}
      value={mappedValue}
      onChange={({ target: { value: v } }) => handleChange(v)}
    />
  );
};

export default ErrorSampler;
