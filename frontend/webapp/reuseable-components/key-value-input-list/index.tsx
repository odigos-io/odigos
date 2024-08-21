import Image from 'next/image';
import React, { useState, useEffect, useRef } from 'react';
import styled from 'styled-components';
import { Input } from '../input';
import { Button } from '../button';
import { Text } from '../text';
import { Tooltip } from '../tooltip';

interface KeyValueInputsListProps {
  initialKeyValuePairs?: { key: string; value: string }[];
  title?: string;
  tooltip?: string;
  onChange?: (validKeyValuePairs: { key: string; value: string }[]) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
`;

const HeaderWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 6px;
`;

const Row = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 12px;
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
  margin-bottom: 4px;
`;

export const KeyValueInputsList: React.FC<KeyValueInputsListProps> = ({
  initialKeyValuePairs = [{ key: '', value: '' }],
  title,
  tooltip,
  onChange,
}) => {
  const [keyValuePairs, setKeyValuePairs] =
    useState<{ key: string; value: string }[]>(initialKeyValuePairs);

  const validPairsRef = useRef<{ key: string; value: string }[]>([]);

  useEffect(() => {
    // Filter out rows where either key or value is empty
    const validKeyValuePairs = keyValuePairs.filter(
      (pair) => pair.key.trim() !== '' && pair.value.trim() !== ''
    );

    // Only trigger onChange if valid key-value pairs have changed
    if (
      JSON.stringify(validPairsRef.current) !==
      JSON.stringify(validKeyValuePairs)
    ) {
      validPairsRef.current = validKeyValuePairs;
      if (onChange) {
        onChange(validKeyValuePairs);
      }
    }
  }, [keyValuePairs, onChange]);

  const handleAddRow = () => {
    setKeyValuePairs([...keyValuePairs, { key: '', value: '' }]);
  };

  const handleDeleteRow = (index: number) => {
    setKeyValuePairs(keyValuePairs.filter((_, i) => i !== index));
  };

  const handleKeyChange = (key: string, index: number) => {
    const newKeyValuePairs = [...keyValuePairs];
    newKeyValuePairs[index].key = key;
    setKeyValuePairs(newKeyValuePairs);
  };

  const handleValueChange = (value: string, index: number) => {
    const newKeyValuePairs = [...keyValuePairs];
    newKeyValuePairs[index].value = value;
    setKeyValuePairs(newKeyValuePairs);
  };

  // Check if any key or value field is empty
  const isAddButtonDisabled = keyValuePairs.some(
    (pair) => pair.key.trim() === '' || pair.value.trim() === ''
  );

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
      {keyValuePairs.map((pair, index) => (
        <Row key={index}>
          <Input
            value={pair.key}
            onChange={(e) => handleKeyChange(e.target.value, index)}
            placeholder="Define attribute"
          />
          <Image
            src="/icons/common/arrow-right.svg"
            alt="Arrow"
            width={16}
            height={16}
          />
          <Input
            value={pair.value}
            onChange={(e) => handleValueChange(e.target.value, index)}
            placeholder="Define value"
          />
          <DeleteButton onClick={() => handleDeleteRow(index)}>
            <Image
              src="/icons/common/trash.svg"
              alt="Delete"
              width={16}
              height={16}
            />
          </DeleteButton>
        </Row>
      ))}
      <AddButton
        disabled={isAddButtonDisabled}
        variant={'tertiary'}
        onClick={handleAddRow}
      >
        <Image src="/icons/common/plus.svg" alt="Add" width={16} height={16} />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export default KeyValueInputsList;
