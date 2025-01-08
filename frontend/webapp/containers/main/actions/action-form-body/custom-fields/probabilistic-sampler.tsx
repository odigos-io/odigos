import React, { useEffect, useMemo } from 'react';
import { Input } from '@/reuseable-components';
import { isEmpty, safeJsonParse } from '@/utils';
import type { ProbabilisticSamplerSpec } from '@/types';

type Props = {
  value: string;
  setValue: (value: string) => void;
  errorMessage?: string;
};

type Parsed = ProbabilisticSamplerSpec;

const MIN = 0,
  MAX = 100;

const ProbabilisticSampler: React.FC<Props> = ({ value, setValue, errorMessage }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { sampling_percentage: '0' }).sampling_percentage, [value]);

  const handleChange = (val: string) => {
    const num = Math.max(MIN, Math.min(Number(val), MAX)) || MIN;

    const payload: Parsed = {
      sampling_percentage: String(num),
    };

    const str = isEmpty(payload.sampling_percentage) ? '' : JSON.stringify(payload);

    setValue(str);
  };

  useEffect(() => {
    if (isEmpty(safeJsonParse(value, {}))) handleChange('0');
  }, [value]);

  return <Input title='Sampling percentage' required type='number' min={MIN} max={MAX} value={mappedValue} onChange={({ target: { value: v } }) => handleChange(v)} errorMessage={errorMessage} />;
};

export default ProbabilisticSampler;
