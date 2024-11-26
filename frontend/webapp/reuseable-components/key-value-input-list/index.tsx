import Image from 'next/image';
import { Text } from '../text';
import { Input } from '../input';
import { Button } from '../button';
import styled from 'styled-components';
import { FieldLabel } from '../field-label';
import React, { useState, useEffect, useRef, useMemo } from 'react';

type Row = {
  key: string;
  value: string;
};

interface KeyValueInputsListProps {
  initialKeyValuePairs?: Row[];
  value?: Row[];
  onChange?: (validKeyValuePairs: Row[]) => void;
  title?: string;
  tooltip?: string;
  required?: boolean;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
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
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
`;

const AddButton = styled(Button)`
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

const INITIAL_ROW: Row = {
  key: '',
  value: '',
};

export const KeyValueInputsList: React.FC<KeyValueInputsListProps> = ({ initialKeyValuePairs = [], value, onChange, title, tooltip, required }) => {
  const [rows, setRows] = useState<Row[]>(value || initialKeyValuePairs);

  useEffect(() => {
    if (!rows.length) setRows([{ ...INITIAL_ROW }]);
  }, []);

  // Filter out rows where either key or value is empty
  const validRows = useMemo(() => rows.filter(({ key, value }) => !!key.trim() && !!value.trim()), [rows]);
  const recordedRows = useRef(JSON.stringify(validRows));

  useEffect(() => {
    const stringified = JSON.stringify(validRows);

    // Only trigger onChange if valid key-value pairs have changed
    if (recordedRows.current !== stringified) {
      recordedRows.current = stringified;

      if (onChange) onChange(validRows);
    }
  }, [validRows, onChange]);

  const handleAddRow = () => {
    setRows((prev) => {
      const payload = [...prev];
      payload.push({ ...INITIAL_ROW });
      return payload;
    });
  };

  const handleDeleteRow = (idx: number) => {
    setRows((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleChange = (key: 'key' | 'value', val: string, idx: number) => {
    setRows((prev) => {
      const payload = [...prev];
      payload[idx][key] = val;
      return payload;
    });
  };

  // Check if any key or value field is empty
  const isAddButtonDisabled = rows.some((pair) => pair.key.trim() === '' || pair.value.trim() === '');
  const isDelButtonDisabled = rows.length <= 1;

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      {rows.map((pair, idx) => (
        <Row key={`key-value-input-list-${idx}`}>
          <Input placeholder='Attribute name' value={pair.key} onChange={(e) => handleChange('key', e.target.value, idx)} autoFocus={rows.length > 1 && idx === rows.length - 1} />
          <Image src='/icons/common/arrow-right.svg' alt='Arrow' width={16} height={16} />
          <Input placeholder='Attribute value' value={pair.value} onChange={(e) => handleChange('value', e.target.value, idx)} />
          <DeleteButton disabled={isDelButtonDisabled} onClick={() => handleDeleteRow(idx)}>
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
