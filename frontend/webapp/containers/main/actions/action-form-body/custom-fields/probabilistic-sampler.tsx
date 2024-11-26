import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { Input } from '@/reuseable-components';
import type { ProbabilisticSamplerSpec } from '@/types';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = ProbabilisticSamplerSpec;

const MIN = 0,
  MAX = 100;

const ProbabilisticSampler: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { sampling_percentage: '0' }).sampling_percentage, [value]);

  const handleChange = (val: string) => {
    const num = Math.max(MIN, Math.min(Number(val), MAX)) || MIN;

    const payload: Parsed = {
      sampling_percentage: String(num),
    };

    const str = !!payload.sampling_percentage ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return (
    <Input
      title='Sampling percentage'
      required
      type='number'
      min={MIN}
      max={MAX}
      value={mappedValue}
      onChange={({ target: { value: v } }) => handleChange(v)}
    />
  );
};

export default ProbabilisticSampler;
