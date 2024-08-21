import Image from 'next/image';
import React, { useState } from 'react';
import styled from 'styled-components';
import { Input } from '../input';
import { Button } from '../button';
import { Text } from '../text';
import { Tooltip } from '../tooltip';

interface InputListProps {
  initialValues?: string[];
  title?: string;
  tooltip?: string;
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

const AddButton = styled(Button)`
  color: white;
  background: transparent;
  display: flex;
  gap: 8px;
  border: none;
  cursor: pointer;
  align-self: flex-start;
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
  margin-bottom: 4px;
`;

const HeaderWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
`;

const InputList: React.FC<InputListProps> = ({
  initialValues = [''],
  title,
  tooltip,
}) => {
  const [inputs, setInputs] = useState<string[]>(initialValues);

  const handleAddInput = () => {
    setInputs([...inputs, '']);
  };

  const handleDeleteInput = (index: number) => {
    setInputs(inputs.filter((_, i) => i !== index));
  };

  const handleInputChange = (value: string, index: number) => {
    const newInputs = [...inputs];
    newInputs[index] = value;
    setInputs(newInputs);
  };

  return (
    <Container>
      {title && (
        <Tooltip text={tooltip || ''}>
          <HeaderWrapper>
            <Title>{title}</Title>
            {tooltip && (
              <Image
                src="/icons/common/info.svg"
                alt=""
                width={16}
                height={16}
                style={{ marginBottom: 4 }}
              />
            )}
          </HeaderWrapper>
        </Tooltip>
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
      <AddButton variant={'tertiary'} onClick={handleAddInput}>
        <Image
          src="/icons/common/plus.svg"
          alt="Delete"
          width={16}
          height={16}
        />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export default InputList;
