import { safeJsonParse } from '@/utils';
import React, { useEffect, useMemo, useState } from 'react';
import { Checkbox } from '@/reuseable-components';
import { FieldTitle, FieldWrapper } from './styled';
import styled from 'styled-components';

type Props = {
  value: string;
  setValue: (value: string) => void;
};

type Parsed = {
  piiCategories: string[];
};

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

const isSelected = (id: string, selected: string[]) => {
  return !!selected?.find((str) => str === id);
};

const PiiMasking: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(() => safeJsonParse<Parsed>(value, { piiCategories: [] }).piiCategories, [value]);

  const [isLastSelection, setIsLastSelection] = useState(false);

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

    setValue(JSON.stringify(payload));
    setIsLastSelection(arr.length === 1);
  };

  return (
    <FieldWrapper>
      <FieldTitle>Attributes to mask</FieldTitle>

      <ListContainer>
        {strictPicklist.map(({ id, label }) => {
          const selected = isSelected(id, mappedValue);

          return (
            <Checkbox
              key={id}
              title={label}
              disabled={isLastSelection && selected}
              initialValue={mappedValue.includes(id)}
              onChange={(bool) => handleChange(id, bool)}
            />
          );
        })}
      </ListContainer>
    </FieldWrapper>
  );
};

export default PiiMasking;
