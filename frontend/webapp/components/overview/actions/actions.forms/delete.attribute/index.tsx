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
}
const ACTION_DATA_KEY = 'actionData';
export function DeleteAttributesForm({
  data,
  onChange,
}: DeleteAttributesProps): React.JSX.Element {
  function handleOnChange(attributeNamesToDelete: string[]): void {
    onChange(ACTION_DATA_KEY, {
      attributeNamesToDelete,
    });
  }

  return (
    <>
      <FormWrapper>
        <MultiInputTable
          placeholder="Add attribute names to delete"
          required
          title="Attribute Names to Delete"
          values={
            data?.attributeNamesToDelete?.length > 0
              ? data.attributeNamesToDelete
              : ['']
          }
          onValuesChange={handleOnChange}
        />
      </FormWrapper>
    </>
  );
}
