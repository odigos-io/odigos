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

interface RenameAttributesProps {
  data: RenameAttributes;
  onChange: (key: string, value: RenameAttributes) => void;
  setIsFormValid?: (value: boolean) => void;
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
  setIsFormValid = () => {},
}: RenameAttributesProps): React.JSX.Element {
  const [keyValuePairs, setKeyValuePairs] = React.useState<KeyValue[]>([]);

  useEffect(() => {
    buildKeyValuePairs();
  }, [data]);

  useEffect(() => {
    validateForm();
  }, [keyValuePairs]);

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    const renames: {
      [key: string]: string;
    } = {};
    keyValues.forEach((keyValue) => {
      renames[keyValue.key] = keyValue.value;
    });

    onChange(ACTION_DATA_KEY, { renames });
    setKeyValuePairs(keyValues); // Update state with new key-value pairs
  }

  function buildKeyValuePairs() {
    if (!data?.renames) {
      setKeyValuePairs(DEFAULT_KEY_VALUE_PAIR);
      return;
    }

    const values = Object.entries(data.renames).map(([key, value], index) => ({
      id: index,
      key,
      value,
    }));

    setKeyValuePairs(values || DEFAULT_KEY_VALUE_PAIR);
  }

  function validateForm() {
    const isValid = keyValuePairs.every(
      (pair) => pair.key.trim() !== '' && pair.value.trim() !== ''
    );
    setIsFormValid(isValid);
  }

  return (
    <>
      <FormWrapper>
        <KeyValuePair
          title="Attributes To Rename  *"
          titleKey="Original Attribute"
          titleValue="New Attribute"
          titleButton="Add Attribute"
          keyValues={keyValuePairs}
          setKeyValues={handleKeyValuesChange}
        />
      </FormWrapper>
    </>
  );
}
