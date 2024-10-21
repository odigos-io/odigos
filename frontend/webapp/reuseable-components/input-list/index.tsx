import Image from 'next/image';
import React, { useEffect, useState } from 'react';
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

const InputList: React.FC<InputListProps> = ({
  initialValues = [''],
  title,
  tooltip,
  required,
  onChange,
  value = [''],
}) => {
  const [inputs, setInputs] = useState<string[]>(value || initialValues);

  useEffect(() => {
    if (initialValues.length > 0) {
      onChange(initialValues);
    }
  }, [initialValues]);

  const handleAddInput = () => {
    setInputs([...inputs, '']);
  };

  const handleDeleteInput = (index: number) => {
    const newInputs = inputs.filter((_, i) => i !== index);
    setInputs(newInputs);
    onChange(newInputs);
  };

  const handleInputChange = (value: string, index: number) => {
    const newInputs = [...inputs];
    newInputs[index] = value;
    setInputs(newInputs);
    onChange(newInputs);
  };

  // Check if any input field is empty
  const isAddButtonDisabled = inputs.some((input) => input.trim() === '');

  return (
    <Container>
      {title && (
        <HeaderWrapper>
          <Title>{title}</Title>
          {!required && (
            <Text color="#7A7A7A" size={14} weight={300} opacity={0.8}>
              (optional)
            </Text>
          )}
          {tooltip && (
            <Tooltip text={tooltip || ''}>
              <Image
                src="/icons/common/info.svg"
                alt=""
                width={16}
                height={16}
                style={{ marginBottom: 4 }}
              />
            </Tooltip>
          )}
        </HeaderWrapper>
      )}
      {inputs.map((value, index) => (
        <InputRow key={index}>
          <Input
            value={value}
            onChange={(e) => handleInputChange(e.target.value, index)}
          />
          <DeleteButton onClick={() => handleDeleteInput(index)}>
            <Image
              src="/icons/common/trash.svg"
              alt="Delete"
              width={16}
              height={16}
            />
          </DeleteButton>
        </InputRow>
      ))}
      <AddButton
        disabled={isAddButtonDisabled}
        variant={'tertiary'}
        onClick={handleAddInput}
      >
        <Image src="/icons/common/plus.svg" alt="Add" width={16} height={16} />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export { InputList };
