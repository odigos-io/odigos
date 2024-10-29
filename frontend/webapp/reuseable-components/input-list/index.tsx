import Image from 'next/image';
import React, { useEffect, useRef, useState } from 'react';
import styled from 'styled-components';
import { Input } from '../input';
import { Button } from '../button';
import { Text } from '../text';
import { Tooltip } from '../tooltip';

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

const Title = styled(Text)`
  font-size: 14px;
  opacity: 0.8;
  line-height: 22px;
`;

const HeaderWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
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
      {title && (
        <HeaderWrapper>
          <Title>{title}</Title>
          {!required && (
            <Text color='#7A7A7A' size={14} weight={300} opacity={0.8}>
              (optional)
            </Text>
          )}
          {tooltip && (
            <Tooltip text={tooltip || ''}>
              <Image src='/icons/common/info.svg' alt='' width={16} height={16} style={{ marginBottom: 4 }} />
            </Tooltip>
          )}
        </HeaderWrapper>
      )}
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
