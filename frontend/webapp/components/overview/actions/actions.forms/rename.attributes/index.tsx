import React, { useEffect } from 'react';
import styled from 'styled-components';
import { KeyValuePair } from '@/design.system';
import { KeyValue } from '@keyval-dev/design-system';
const FormWrapper = styled.div`
  width: 375px;
`;

interface RenameAttributes {
  renames: {
    [key: string]: string;
  };
}

interface DeleteAttributesProps {
  data: RenameAttributes;
  onChange: (key: string, value: RenameAttributes) => void;
}
const DEFAULT_KEY_VALUE_PAIR = [
  {
    id: 0,
    key: '',
    value: '',
  },
];

const ACTION_DATA_KEY = 'actionData';
export function RenameAttributesForm({
  data,
  onChange,
}: DeleteAttributesProps): React.JSX.Element {
  const [keyValuePairs, setKeyValuePairs] = React.useState<KeyValue[]>([]);

  useEffect(() => {
    buildKeyValuePairs();
  }, [data]);

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    const renames: {
      [key: string]: string;
    } = {};
    keyValues.forEach((keyValue) => {
      renames[keyValue.key] = keyValue.value;
    });
    console.log({ object: renames });
    onChange(ACTION_DATA_KEY, { renames });
  }

  function buildKeyValuePairs() {
    console.log({ data });
    if (!data?.renames) {
      setKeyValuePairs(DEFAULT_KEY_VALUE_PAIR);
      return;
    }

    const values = Object.entries(data.renames).map(([key, value], index) => ({
      id: index,
      key,
      value,
    }));
    console.log({ values });
    setKeyValuePairs(values || DEFAULT_KEY_VALUE_PAIR);
  }

  return (
    <>
      <FormWrapper>
        <KeyValuePair
          title="Cluster Attributes *"
          titleKey="Attribute"
          titleButton="Add Attribute"
          keyValues={keyValuePairs}
          setKeyValues={handleKeyValuesChange}
        />
      </FormWrapper>
    </>
  );
}
