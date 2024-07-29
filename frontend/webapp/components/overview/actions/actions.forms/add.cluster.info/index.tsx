import React, { useEffect } from 'react';
import styled from 'styled-components';
import { KeyValuePair } from '@/design.system';
import { KeyValue } from '@keyval-dev/design-system';

const DEFAULT_KEY_VALUE_PAIR = [
  {
    id: 0,
    key: '',
    value: '',
  },
];

const FormWrapper = styled.div`
  width: 375px;
`;

interface ClusterAttributes {
  clusterAttributes: {
    attributeName: string;
    attributeStringValue: string;
  }[];
}

interface AddClusterInfoFormProps {
  data: ClusterAttributes | null;
  onChange: (key: string, keyValues: ClusterAttributes | null) => void;
  setIsFormValid?: (value: boolean) => void;
}

const ACTION_DATA_KEY = 'actionData';

export function AddClusterInfoForm({
  data,
  onChange,
  setIsFormValid = () => {},
}: AddClusterInfoFormProps): React.JSX.Element {
  const [keyValuePairs, setKeyValuePairs] = React.useState<KeyValue[]>([]);

  useEffect(() => {
    buildKeyValuePairs();
  }, [data]);

  useEffect(() => {
    validateForm();
  }, [keyValuePairs]);

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    const actionData = {
      clusterAttributes: keyValues.map((keyValue) => ({
        attributeName: keyValue.key,
        attributeStringValue: keyValue.value,
      })),
    };

    if (
      actionData.clusterAttributes.length === 1 &&
      actionData.clusterAttributes[0].attributeName === '' &&
      actionData.clusterAttributes[0].attributeStringValue === ''
    ) {
      onChange(ACTION_DATA_KEY, null);
    } else {
      onChange(ACTION_DATA_KEY, actionData);
    }

    setKeyValuePairs(keyValues); // Update state with new key-value pairs
  }

  function buildKeyValuePairs() {
    if (data?.clusterAttributes.length === 0) {
      setKeyValuePairs(DEFAULT_KEY_VALUE_PAIR);
    }

    const values = data?.clusterAttributes.map((keyValue, index) => ({
      id: index,
      key: keyValue.attributeName,
      value: keyValue.attributeStringValue,
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
