import React, { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { FieldTitle, FieldWrapper } from './styled';
import { KeyValueInputsList } from '@/reuseable-components';
import type { RenameAttributesSpec } from '@/types';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = RenameAttributesSpec;

const RenameAttributes: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(
    () => Object.entries(safeJsonParse<Parsed>(value, { renames: {} }).renames).map(([k, v]) => ({ key: k, value: v })),
    [value]
  );

  const handleChange = (
    arr: {
      key: string;
      value: string;
    }[]
  ) => {
    const payload: Parsed = {
      renames: {},
    };

    arr.forEach((obj) => {
      payload.renames[obj.key] = obj.value;
    });

    const str = !!arr.length ? JSON.stringify(payload) : '';

    setValue(str);
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to rename</FieldTitle>
      <KeyValueInputsList value={mappedValue} onChange={handleChange} />
    </FieldWrapper>
  );
};

export default RenameAttributes;
