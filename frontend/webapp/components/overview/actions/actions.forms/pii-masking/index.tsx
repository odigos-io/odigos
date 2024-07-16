import { KeyvalCheckbox } from '@/design.system';
import React, { useEffect } from 'react';
import styled from 'styled-components';

const piiCategoriesCheckbox = [
  {
    id: 'CREDIT_CARD',
    label: 'Credit Card',
  },
];

const FormWrapper = styled.div`
  width: 375px;
`;

interface PiiMasking {
  piiCategories: string[];
}

interface PiiMaskingProps {
  data: PiiMasking;
  onChange: (key: string, value: PiiMasking | null) => void;
  setIsFormValid?: (isValid: boolean) => void;
}
const ACTION_DATA_KEY = 'actionData';
export function PiiMaskingForm({
  data,
  onChange,
  setIsFormValid,
}: PiiMaskingProps): React.JSX.Element {
  useEffect(() => {
    onChange(ACTION_DATA_KEY, {
      piiCategories: ['CREDIT_CARD'],
    });
    setIsFormValid && setIsFormValid(true);
  }, []);

  function handleOnChange(value: string): void {
    let piiCategories: string[] = [];

    if (piiCategories.includes(value)) {
      piiCategories = piiCategories.filter((category) => category !== value);
    } else {
      piiCategories.push(value);
    }

    onChange(ACTION_DATA_KEY, {
      piiCategories,
    });
  }

  return (
    <>
      <FormWrapper>
        {piiCategoriesCheckbox.map((checkbox) => (
          <KeyvalCheckbox
            disabled
            key={checkbox?.id}
            value={data?.piiCategories?.includes(checkbox?.id)}
            onChange={() => handleOnChange(checkbox?.id)}
            label={checkbox?.label}
          />
        ))}
      </FormWrapper>
    </>
  );
}
