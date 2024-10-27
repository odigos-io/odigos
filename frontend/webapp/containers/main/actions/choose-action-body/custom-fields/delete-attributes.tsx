import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { InputList } from '@/reuseable-components';
import { FieldTitle, FieldWrapper } from './styled';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = {
  attributeNamesToDelete: string[];
};

const DeleteAttributes: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { attributeNamesToDelete: [] }).attributeNamesToDelete, [value]);

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
