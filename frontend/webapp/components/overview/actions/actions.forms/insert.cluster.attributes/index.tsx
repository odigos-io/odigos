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

interface InsertClusterAttributesFormProps {
  data: KeyValue[] | null;
  onChange: (
    key: string,
    keyValues: {
      clusterAttributes: {
        attributeName: string;
        attributeStringValue: string;
      }[];
    }
  ) => void;
}

export function InsertClusterAttributesForm({
  data,
  onChange,
}: InsertClusterAttributesFormProps): React.JSX.Element {
  const [keyValues, setKeyValues] = React.useState<KeyValue[]>(
    data || DEFAULT_KEY_VALUE_PAIR
  );

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    setKeyValues(keyValues);

    const data = keyValues.map((keyValue) => {
      return {
        attributeName: keyValue.key,
        attributeStringValue: keyValue.value,
      };
    });
    onChange('actionData', { clusterAttributes: data });
  }

  return (
    <>
      <FormWrapper>
        <KeyValuePair
          title="Insert cluster attributes"
          titleKey="Attribute"
          titleButton="Add Attribute"
          keyValues={keyValues}
          setKeyValues={handleKeyValuesChange}
        />
      </FormWrapper>
    </>
  );
}
