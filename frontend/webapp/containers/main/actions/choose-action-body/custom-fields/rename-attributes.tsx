import React, { useMemo } from 'react';
import styled from 'styled-components';
import { KeyValueInputsList, Text } from '@/reuseable-components';
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
  renames: {
    [oldKey: string]: string;
  };
};

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

    setValue(JSON.stringify(payload));
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to rename</FieldTitle>
      <KeyValueInputsList value={mappedValue} onChange={handleChange} />
    </FieldWrapper>
  );
};

export default RenameAttributes;
