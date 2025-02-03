import React, { useMemo } from 'react';
import type { DeleteAttributesSpec } from '@/types';
import { InputList, safeJsonParse } from '@odigos/ui-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
  errorMessage?: string;
};

type Parsed = DeleteAttributesSpec;

const DeleteAttributes: React.FC<Props> = ({ value, setValue, errorMessage }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { attributeNamesToDelete: [] }).attributeNamesToDelete, [value]);

  const handleChange = (arr: string[]) => {
    const payload: Parsed = {
      attributeNamesToDelete: arr,
    };

    const str = !!payload.attributeNamesToDelete.length ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return <InputList title='Attributes to delete' value={mappedValue} onChange={handleChange} required errorMessage={errorMessage} />;
};

export default DeleteAttributes;
