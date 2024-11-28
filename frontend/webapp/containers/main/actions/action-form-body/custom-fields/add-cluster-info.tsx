import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import type { AddClusterInfoSpec } from '@/types';
import { KeyValueInputsList } from '@/reuseable-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
  errorMessage?: string;
};

type Parsed = AddClusterInfoSpec;

const AddClusterInfo: React.FC<Props> = ({ value, setValue, errorMessage }) => {
  const mappedValue = useMemo(
    () =>
      safeJsonParse<Parsed>(value, { clusterAttributes: [] }).clusterAttributes.map((obj) => ({
        key: obj.attributeName,
        value: obj.attributeStringValue,
      })),
    [value],
  );

  const handleChange = (arr: { key: string; value: string }[]) => {
    const payload: Parsed = {
      clusterAttributes: arr.map((obj) => ({
        attributeName: obj.key,
        attributeStringValue: obj.value,
      })),
    };

    const str = !!payload.clusterAttributes.length ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return <KeyValueInputsList title='Resource Attributes' value={mappedValue} onChange={handleChange} required errorMessage={errorMessage} />;
};

export default AddClusterInfo;
