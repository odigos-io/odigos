import React, { useEffect, useMemo, useState } from 'react';
import { safeJsonParse } from '@/utils';
import type { PiiMaskingSpec } from '@/types';
import styled, { css } from 'styled-components';
import { Checkbox, FieldError, FieldLabel } from '@/reuseable-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
  errorMessage?: string;
};

type Parsed = PiiMaskingSpec;

const ListContainer = styled.div<{ $hasError: boolean }>`
  display: flex;
  flex-direction: row;
  gap: 32px;
  ${({ $hasError }) =>
    $hasError &&
    css`
      border: 1px solid ${({ theme }) => theme.text.error};
      border-radius: 32px;
      padding: 8px;
    `}
`;

const strictPicklist = [
  {
    id: 'CREDIT_CARD',
    label: 'Credit Card',
  },
];

const PiiMasking: React.FC<Props> = ({ value, setValue, errorMessage }) => {
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
    <div>
      <FieldLabel title='Attributes to mask' required />
      <ListContainer $hasError={!!errorMessage}>
        {strictPicklist.map(({ id, label }) => (
          <Checkbox key={id} title={label} disabled={isLastSelection && mappedValue.includes(id)} value={mappedValue.includes(id)} onChange={(bool) => handleChange(id, bool)} />
        ))}
      </ListContainer>
      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </div>
  );
};

export default PiiMasking;
