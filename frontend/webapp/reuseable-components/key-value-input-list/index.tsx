import Image from 'next/image';
import React, { useState, useEffect, useRef } from 'react';
import styled from 'styled-components';
import { Input } from '../input';
import { Button } from '../button';
import { Text } from '../text';
import { Tooltip } from '../tooltip';

interface KeyValueInputsListProps {
  initialKeyValuePairs?: { key: string; value: string }[];
  value?: { key: string; value: string }[];
  title?: string;
  tooltip?: string;
  required?: boolean;
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
  margin-bottom: 4px;
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
`;

const INITIAL = [{ key: '', value: '' }];

export const KeyValueInputsList: React.FC<KeyValueInputsListProps> = ({
  initialKeyValuePairs = INITIAL,
  value = INITIAL,
  onChange,
  title,
  tooltip,
  required,
}) => {
  const [keyValuePairs, setKeyValuePairs] = useState<{ key: string; value: string }[]>(value || initialKeyValuePairs);

  useEffect(() => {
    if (!keyValuePairs.length) setKeyValuePairs(INITIAL);
  }, []);

  const recordedPairs = useRef('');

  useEffect(() => {
    // Filter out rows where either key or value is empty
    const validKeyValuePairs = keyValuePairs.filter((pair) => pair.key.trim() !== '' && pair.value.trim() !== '');
    const stringified = JSON.stringify(validKeyValuePairs);

    // Only trigger onChange if valid key-value pairs have changed
    if (recordedPairs.current !== stringified) {
      recordedPairs.current = stringified;

      if (onChange) onChange(validKeyValuePairs);
    }
  }, [keyValuePairs, onChange]);

  const handleAddRow = () => {
    setKeyValuePairs((prev) => {
      const payload = [...prev];
      payload.push({ key: '', value: '' });
      return payload;
    });
  };

  const handleDeleteRow = (idx: number) => {
    setKeyValuePairs((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleChange = (key: 'key' | 'value', val: string, idx: number) => {
    setKeyValuePairs((prev) => {
      const payload = [...prev];
      payload[idx][key] = val;
      return payload;
    });
  };

  // Check if any key or value field is empty
  const isAddButtonDisabled = keyValuePairs.some((pair) => pair.key.trim() === '' || pair.value.trim() === '');

  return (
    <Container>
      {title && (
        <Tooltip text={tooltip || ''}>
          <HeaderWrapper>
            <Title>{title}</Title>
            {!required && (
              <Text color='#7A7A7A' size={14} weight={300} opacity={0.8}>
                (optional)
              </Text>
            )}
            {tooltip && <Image src='/icons/common/info.svg' alt='' width={16} height={16} style={{ marginBottom: 4 }} />}
          </HeaderWrapper>
        </Tooltip>
      )}

      {keyValuePairs.map((pair, index) => (
        <Row key={`key-value-pair-${title}-${index}`}>
          <Input value={pair.key} onChange={(e) => handleChange('key', e.target.value, index)} placeholder='Define attribute' />
          <Image src='/icons/common/arrow-right.svg' alt='Arrow' width={16} height={16} />
          <Input value={pair.value} onChange={(e) => handleChange('value', e.target.value, index)} placeholder='Define value' />
          <DeleteButton onClick={() => handleDeleteRow(index)}>
            <Image src='/icons/common/trash.svg' alt='Delete' width={16} height={16} />
          </DeleteButton>
        </Row>
      ))}
      <AddButton disabled={isAddButtonDisabled} variant={'tertiary'} onClick={handleAddRow}>
        <Image src='/icons/common/plus.svg' alt='Add' width={16} height={16} />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export default KeyValueInputsList;
