import styled from 'styled-components';
import { safeJsonParse } from '@/utils';
import type { PiiMaskingSpec } from '@/types';
import React, { useEffect, useMemo, useState } from 'react';
import { Checkbox, FieldLabel } from '@/reuseable-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = PiiMaskingSpec;

const ListContainer = styled.div`
  display: flex;
  flex-direction: row;
  gap: 32px;
`;

const strictPicklist = [
  {
    id: 'CREDIT_CARD',
    label: 'Credit Card',
  },
];

const PiiMasking: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { piiCategories: [] }).piiCategories, [value]);
  const [isLastSelection, setIsLastSelection] = useState(mappedValue.length === 1);

  useEffect(() => {
    if (!mappedValue.length) {
      const payload: Parsed = {
        piiCategories: strictPicklist.map(({ id }) => id),
      };

      setValue(JSON.stringify(payload));
      setIsLastSelection(payload.piiCategories.length === 1);
    }
    // eslint-disable-next-line
  }, []);

  const handleChange = (id: string, isAdd: boolean) => {
    const arr = isAdd ? [...mappedValue, id] : mappedValue.filter((str) => str !== id);

    const payload: Parsed = {
      piiCategories: arr,
    };

    const str = !!arr.length ? JSON.stringify(payload) : '';

    setValue(str);
    setIsLastSelection(arr.length === 1);
  };

  return (
    <>
      <FieldLabel title='Attributes to mask' required />

      <ListContainer>
        {strictPicklist.map(({ id, label }) => (
          <Checkbox
            key={id}
            title={label}
            disabled={isLastSelection && mappedValue.includes(id)}
            initialValue={mappedValue.includes(id)}
            onChange={(bool) => handleChange(id, bool)}
          />
        ))}
      </ListContainer>
    </>
  );
};

export default PiiMasking;
