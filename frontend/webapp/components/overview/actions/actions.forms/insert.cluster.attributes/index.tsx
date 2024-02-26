import React from 'react';
import styled from 'styled-components';
import { KeyValuePair, KeyvalText } from '@/design.system';
import { KeyValue } from '@keyval-dev/design-system';
import theme from '@/styles/palette';

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
  onChange: (keyValues: KeyValue[]) => void;
}

export function InsertClusterAttributesForm({
  onChange,
}: InsertClusterAttributesFormProps): React.JSX.Element {
  const [keyValues, setKeyValues] = React.useState<KeyValue[]>(
    DEFAULT_KEY_VALUE_PAIR
  );

  function handleKeyValuesChange(keyValues: KeyValue[]): void {
    setKeyValues(keyValues);
    onChange(keyValues);
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
