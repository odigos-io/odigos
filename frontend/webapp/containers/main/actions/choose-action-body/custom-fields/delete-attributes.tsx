import React, { useMemo } from 'react';
import styled from 'styled-components';
import { InputList, Text } from '@/reuseable-components';

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
  attributeNamesToDelete: string[];
};

const DeleteAttributes: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => (value ? (JSON.parse(value) as Parsed).attributeNamesToDelete : undefined), [value]);

  const handleChange = (arr: string[]) => {
    const payload: Parsed = {
      attributeNamesToDelete: arr,
    };

    setValue(JSON.stringify(payload));
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to delete</FieldTitle>
      <InputList value={mappedValue} onChange={handleChange} />
    </FieldWrapper>
  );
};

export default DeleteAttributes;
