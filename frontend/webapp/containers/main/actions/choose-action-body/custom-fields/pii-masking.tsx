import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { InputList } from '@/reuseable-components';
import { FieldTitle, FieldWrapper } from './styled';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = {
  piiCategories: string[];
};

const PiiMasking: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { piiCategories: [] }).piiCategories, [value]);

  const handleChange = (arr: string[]) => {
    const payload: Parsed = {
      piiCategories: arr,
    };

    setValue(JSON.stringify(payload));
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to mask</FieldTitle>
      <InputList value={mappedValue} onChange={handleChange} />
    </FieldWrapper>
  );
};

export default PiiMasking;
