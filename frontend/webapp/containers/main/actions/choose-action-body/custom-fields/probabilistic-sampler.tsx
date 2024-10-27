import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { Input } from '@/reuseable-components';
import { FieldTitle, FieldWrapper } from './styled';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = {
  sampling_percentage: string;
};

const MIN = 0,
  MAX = 100;

const ProbabilisticSampler: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { sampling_percentage: '0' }).sampling_percentage, [value]);

  const handleChange = (val: string) => {
    let num = Number(val);

    if (Number.isNaN(num) || num < MIN || num > MAX) {
      num = MIN;
    }

    const payload: Parsed = {
      sampling_percentage: String(num),
    };

    setValue(JSON.stringify(payload));
  };

  return (
    <FieldWrapper>
      <FieldTitle>Sampling percentage</FieldTitle>
      <Input type='number' min={MIN} max={MAX} value={mappedValue} onChange={({ target: { value: v } }) => handleChange(v)} />
    </FieldWrapper>
  );
};

export default ProbabilisticSampler;
