import styled from 'styled-components';
import type { InstrumentationRuleInput } from '@/types';
import React, { useEffect, useMemo, useState } from 'react';
import { Checkbox, FieldLabel } from '@/reuseable-components';

type Props = {
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
};

type Parsed = InstrumentationRuleInput['payloadCollection'];

const ListContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
`;

const strictPicklist = [
  {
    id: 'httpRequest',
    label: 'Collect HTTP Request',
  },
  {
    id: 'httpResponse',
    label: 'Collect HTTP Response',
  },
  {
    id: 'dbQuery',
    label: 'Collect DB Query',
  },
  {
    id: 'messaging',
    label: 'Collect Messaging',
  },
];

const PayloadCollection: React.FC<Props> = ({ value, setValue }) => {
  const mappedValue = useMemo(
    () =>
      Object.entries(value.payloadCollection)
        .filter(([k, v]) => !!v)
        .map(([k]) => k),
    [value]
  );

  const [isLastSelection, setIsLastSelection] = useState(mappedValue.length === 1);

  useEffect(() => {
    if (!mappedValue.length) {
      const payload: Parsed = {
        httpRequest: {},
        httpResponse: {},
        dbQuery: {},
        messaging: {},
      };

      setValue('payloadCollection', payload);
      setIsLastSelection(false);
    }
    // eslint-disable-next-line
  }, []);

  const handleChange = (id: string, isAdd: boolean) => {
    const arr = isAdd ? [...mappedValue, id] : mappedValue.filter((str) => str !== id);

    const payload: Parsed = {
      httpRequest: arr.includes('httpRequest') ? {} : null,
      httpResponse: arr.includes('httpResponse') ? {} : null,
      dbQuery: arr.includes('dbQuery') ? {} : null,
      messaging: arr.includes('messaging') ? {} : null,
    };

    setValue('payloadCollection', payload);
    setIsLastSelection(arr.length === 1);
  };

  return (
    <>
      <FieldLabel title='Type of data to collect' required />

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

export default PayloadCollection;
