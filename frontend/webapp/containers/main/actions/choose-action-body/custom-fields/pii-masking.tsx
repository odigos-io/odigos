import React, { useMemo } from 'react';
import styled from 'styled-components';
import { InputList, Text } from '@/reuseable-components';
import { safeJsonParse } from '@/utils';

const FieldWrapper = styled.div`
  width: 100%;
  margin: 8px 0;
`;

const FieldTitle = styled(Text)`
  margin-bottom: 12px;
`;

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
