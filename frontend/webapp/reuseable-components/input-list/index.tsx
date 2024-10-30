import Image from 'next/image';
import { Text } from '../text';
import { Input } from '../input';
import { Button } from '../button';
import styled from 'styled-components';
import { FieldLabel } from '../field-label';
import React, { useEffect, useRef, useState } from 'react';

interface InputListProps {
  initialValues?: string[];
  title?: string;
  tooltip?: string;
  required?: boolean;
  value?: string[];
  onChange: (values: string[]) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
`;

const InputRow = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
`;

const DeleteButton = styled.button`
  background: none;
  border: none;
  cursor: pointer;
`;

const AddButton = styled(Button)<{ disabled: boolean }>`
  color: white;
  background: transparent;
  display: flex;
  gap: 8px;
  border: none;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  align-self: flex-start;
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
  transition: opacity 0.3s;
`;

const ButtonText = styled(Text)`
  font-size: 14px;
  font-weight: 500;
  font-family: ${({ theme }) => theme.font_family.secondary};
  text-decoration-line: underline;
`;

const INITIAL = [''];

const InputList: React.FC<InputListProps> = ({ initialValues = INITIAL, value = INITIAL, onChange, title, tooltip, required }) => {
  const [inputs, setInputs] = useState<string[]>(value || initialValues);

  useEffect(() => {
    if (!inputs.length) setInputs(INITIAL);
  }, []);

  const recordedValues = useRef('');

  useEffect(() => {
    // Filter out rows where either key or value is empty
    const validValues = inputs.filter((val) => val.trim() !== '');
    const stringified = JSON.stringify(validValues);

    // Only trigger onChange if valid key-value pairs have changed
    if (recordedValues.current !== stringified) {
      recordedValues.current = stringified;

      if (onChange) onChange(validValues);
    }
  }, [inputs, onChange]);

  const handleAddInput = () => {
    setInputs((prev) => {
      const payload = [...prev];
      payload.push('');
      return payload;
    });
  };

  const handleDeleteInput = (idx: number) => {
    setInputs((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleInputChange = (val: string, idx: number) => {
    setInputs((prev) => {
      const payload = [...prev];
      payload[idx] = val;
      return payload;
    });
  };

  // Check if any input field is empty
  const isAddButtonDisabled = inputs.some((input) => input.trim() === '');

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      {inputs.map((value, index) => (
        <InputRow key={index}>
          <Input value={value} onChange={(e) => handleInputChange(e.target.value, index)} />
          <DeleteButton onClick={() => handleDeleteInput(index)}>
            <Image src='/icons/common/trash.svg' alt='Delete' width={16} height={16} />
          </DeleteButton>
        </InputRow>
      ))}

      <AddButton disabled={isAddButtonDisabled} variant={'tertiary'} onClick={handleAddInput}>
        <Image src='/icons/common/plus.svg' alt='Add' width={16} height={16} />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export { InputList };
