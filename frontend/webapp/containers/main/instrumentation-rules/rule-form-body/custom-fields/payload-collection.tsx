import React, { useEffect, useMemo, useState } from 'react';
import styled, { css } from 'styled-components';
import { Checkbox, FieldError, FieldLabel } from '@/reuseable-components';
import { PayloadCollectionType, type InstrumentationRuleInput } from '@/types';

type Props = {
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
  formErrors: Record<string, string>;
};

type Parsed = InstrumentationRuleInput['payloadCollection'];

const ListContainer = styled.div<{ $hasError: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
  ${({ $hasError }) =>
    $hasError &&
    css`
      border: 1px solid red;
      border-radius: 16px;
      padding: 8px;
    `}
`;

const strictPicklist = [
  {
    id: PayloadCollectionType.HTTP_REQUEST,
    label: 'Collect HTTP Request',
  },
  {
    id: PayloadCollectionType.HTTP_RESPONSE,
    label: 'Collect HTTP Response',
  },
  {
    id: PayloadCollectionType.DB_QUERY,
    label: 'Collect DB Query',
  },
  {
    id: PayloadCollectionType.MESSAGING,
    label: 'Collect Messaging',
  },
];

const PayloadCollection: React.FC<Props> = ({ value, setValue, formErrors }) => {
  const errorMessage = formErrors['payloadCollection'];

  const mappedValue = useMemo(
    () =>
      Object.entries(value['payloadCollection'])
        .filter(([k, v]) => !!v)
        .map(([k]) => k),
    [value],
  );

  const [isLastSelection, setIsLastSelection] = useState(mappedValue.length === 1);

  useEffect(() => {
    if (!mappedValue.length) {
      const payload: Parsed = {
        [PayloadCollectionType.HTTP_REQUEST]: {},
        [PayloadCollectionType.HTTP_RESPONSE]: {},
        [PayloadCollectionType.DB_QUERY]: {},
        [PayloadCollectionType.MESSAGING]: {},
      };

      setValue('payloadCollection', payload);
      setIsLastSelection(false);
    }
    // eslint-disable-next-line
  }, []);

  const handleChange = (id: string, isAdd: boolean) => {
    const arr = isAdd ? [...mappedValue, id] : mappedValue.filter((str) => str !== id);

    const payload: Parsed = {
      [PayloadCollectionType.HTTP_REQUEST]: arr.includes(PayloadCollectionType.HTTP_REQUEST) ? {} : null,
      [PayloadCollectionType.HTTP_RESPONSE]: arr.includes(PayloadCollectionType.HTTP_RESPONSE) ? {} : null,
      [PayloadCollectionType.DB_QUERY]: arr.includes(PayloadCollectionType.DB_QUERY) ? {} : null,
      [PayloadCollectionType.MESSAGING]: arr.includes(PayloadCollectionType.MESSAGING) ? {} : null,
    };

    setValue('payloadCollection', payload);
    setIsLastSelection(arr.length === 1);
  };

  return (
    <div>
      <FieldLabel title='Type of data to collect' required />
      <ListContainer $hasError={!!errorMessage}>
        {strictPicklist.map(({ id, label }) => (
          <Checkbox key={id} title={label} disabled={isLastSelection && mappedValue.includes(id)} initialValue={mappedValue.includes(id)} onChange={(bool) => handleChange(id, bool)} />
        ))}
      </ListContainer>
      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </div>
  );
};

export default PayloadCollection;
