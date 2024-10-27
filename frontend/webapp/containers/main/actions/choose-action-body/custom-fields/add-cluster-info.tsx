import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { FieldTitle, FieldWrapper } from './styled';
import { KeyValueInputsList } from '@/reuseable-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = {
  clusterAttributes: {
    attributeName: string;
    attributeStringValue: string;
  }[];
};

const AddClusterInfo: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(
    () =>
      safeJsonParse<Parsed>(value, { clusterAttributes: [] }).clusterAttributes.map((obj) => ({
        key: obj.attributeName,
        value: obj.attributeStringValue,
      })),
    [value]
  );

  const handleChange = (
    arr: {
      key: string;
      value: string;
    }[]
  ) => {
    const payload: Parsed = {
      clusterAttributes: arr.map((obj) => ({
        attributeName: obj.key,
        attributeStringValue: obj.value,
      })),
    };

    setValue(JSON.stringify(payload));
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to add</FieldTitle>
      <KeyValueInputsList value={mappedValue} onChange={handleChange} />
    </FieldWrapper>
  );
};

export default AddClusterInfo;