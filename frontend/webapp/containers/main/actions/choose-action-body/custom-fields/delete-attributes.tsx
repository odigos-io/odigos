import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { InputList } from '@/reuseable-components';
import type { DeleteAttributesSpec } from '@/types';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = DeleteAttributesSpec;

const DeleteAttributes: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { attributeNamesToDelete: [] }).attributeNamesToDelete, [value]);

  const handleChange = (arr: string[]) => {
    const payload: Parsed = {
      attributeNamesToDelete: arr,
    };

    const str = !!payload.attributeNamesToDelete.length ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return <InputList title='Attributes to delete' required value={mappedValue} onChange={handleChange} />;
};

export default DeleteAttributes;
