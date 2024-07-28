import React, { useEffect } from 'react';
import styled from 'styled-components';
import { MultiInputTable } from '@/design.system';

const FormWrapper = styled.div`
  width: 375px;
`;

interface DeleteAttributes {
  attributeNamesToDelete: string[];
}

interface DeleteAttributesProps {
  data: DeleteAttributes;
  onChange: (key: string, value: DeleteAttributes | null) => void;
  setIsFormValid?: (value: boolean) => void;
}
const ACTION_DATA_KEY = 'actionData';

export function DeleteAttributesForm({
  data,
  onChange,
  setIsFormValid = () => {},
}: DeleteAttributesProps): React.JSX.Element {
  const [attributeNames, setAttributeNames] = React.useState<string[]>(
    data?.attributeNamesToDelete || ['']
  );

  useEffect(() => {
    validateForm();
  }, [attributeNames]);

  function handleOnChange(attributeNamesToDelete: string[]): void {
    onChange(ACTION_DATA_KEY, {
      attributeNamesToDelete,
    });
    setAttributeNames(attributeNamesToDelete);
  }

  function validateForm() {
    const isValid = attributeNames.every((name) => name.trim() !== '');
    setIsFormValid(isValid);
  }

  return (
    <>
      <FormWrapper>
        <MultiInputTable
          placeholder="Add attribute names to delete"
          required
          title="Attribute Names to Delete"
          values={attributeNames.length > 0 ? attributeNames : ['']}
          onValuesChange={handleOnChange}
        />
      </FormWrapper>
    </>
  );
}
