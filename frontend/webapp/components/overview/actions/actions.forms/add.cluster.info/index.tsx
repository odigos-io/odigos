import React from 'react';
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
}

const ACTION_DATA_KEY = 'actionData';

export function AddClusterInfoForm({
  data,
  onChange,
}: AddClusterInfoFormProps): React.JSX.Element {
  const [keyValues, setKeyValues] = React.useState<ClusterAttributes>(
    data || { clusterAttributes: [] }
  );

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    const actionData = {
      clusterAttributes: keyValues.map((keyValue) => ({
        attributeName: keyValue.key,
        attributeStringValue: keyValue.value,
      })),
    };

    setKeyValues(actionData);
    if (
      actionData.clusterAttributes.length === 1 &&
      actionData.clusterAttributes[0].attributeName === '' &&
      actionData.clusterAttributes[0].attributeStringValue === ''
    ) {
      onChange(ACTION_DATA_KEY, null);
    } else {
      onChange(ACTION_DATA_KEY, actionData);
    }
  }

  function getKeyValuePairs(): KeyValue[] {
    if (keyValues.clusterAttributes.length === 0) {
      return DEFAULT_KEY_VALUE_PAIR;
    }

    return keyValues.clusterAttributes.map((keyValue, index) => ({
      id: index,
      key: keyValue.attributeName,
      value: keyValue.attributeStringValue,
    }));
  }

  return (
    <>
      <FormWrapper>
        <KeyValuePair
          title="Cluster Attributes *"
          titleKey="Attribute"
          titleButton="Add Attribute"
          keyValues={getKeyValuePairs()}
          setKeyValues={handleKeyValuesChange}
        />
      </FormWrapper>
    </>
  );
}
