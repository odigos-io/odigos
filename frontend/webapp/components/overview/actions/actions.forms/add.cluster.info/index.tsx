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

interface AddClusterInfoFormProps {
  data: KeyValue[] | null;
  onChange: (
    key: string,
    keyValues: {
      clusterAttributes:
        | {
            attributeName: string;
            attributeStringValue: string;
          }[];
    } | null
  ) => void;
}

export function AddClusterInfoForm({
  data,
  onChange,
}: AddClusterInfoFormProps): React.JSX.Element {
  const [keyValues, setKeyValues] = React.useState<KeyValue[]>(
    data || DEFAULT_KEY_VALUE_PAIR
  );

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    setKeyValues(keyValues);

    let newData = keyValues.map((keyValue) => ({
      attributeName: keyValue.key,
      attributeStringValue: keyValue.value,
    }));

    // Set newData to null if it meets the condition to indicate a "false" value
    if (
      newData.length === 1 &&
      newData[0].attributeName === '' &&
      newData[0].attributeStringValue === ''
    ) {
      onChange('actionData', null);
    } else {
      onChange('actionData', { clusterAttributes: newData });
    }
  }

  return (
    <>
      <FormWrapper>
        <KeyValuePair
          title="Cluster Attributes *"
          titleKey="Attribute"
          titleButton="Add Attribute"
          keyValues={keyValues}
          setKeyValues={handleKeyValuesChange}
        />
      </FormWrapper>
    </>
  );
}
