import React, { useState, useEffect, useRef, useMemo, type KeyboardEventHandler } from 'react';
import theme from '@/styles/theme';
import styled from 'styled-components';
import { ArrowIcon, PlusIcon, TrashIcon } from '@/assets';
import { Button, FieldError, FieldLabel, Input, Text } from '@/reuseable-components';

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
  errorMessage?: string;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
`;

const ListContainer = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

const RowWrapper = styled.div`
  display: flex;
  align-items: center;
  gap: 16px;
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

export const KeyValueInputsList: React.FC<KeyValueInputsListProps> = ({ initialKeyValuePairs = [], value, onChange, title, tooltip, required, errorMessage }) => {
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

  const handleKeyDown: KeyboardEventHandler<HTMLInputElement> = (e) => {
    e.stopPropagation();
  };

  // Check if any key or value field is empty
  const isAddButtonDisabled = rows.some(({ key, value }) => key.trim() === '' || value.trim() === '');
  const isDelButtonDisabled = rows.length <= 1;

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      <ListContainer>
        {rows.map(({ key, value }, idx) => (
          <RowWrapper key={`key-value-input-list-${idx}`}>
            <Input
              placeholder='Attribute name'
              value={key}
              onChange={(e) => {
                e.stopPropagation();
                handleChange('key', e.target.value, idx);
              }}
              onKeyDown={handleKeyDown}
              hasError={!!errorMessage && (!required || (required && !key))}
              autoFocus={!value && rows.length > 1 && idx === rows.length - 1}
            />
            <div>
              <ArrowIcon rotate={180} fill={theme.text.darker_grey} />
            </div>
            <Input
              placeholder='Attribute value'
              value={value}
              onChange={(e) => {
                e.stopPropagation();
                handleChange('value', e.target.value, idx);
              }}
              onKeyDown={handleKeyDown}
              hasError={!!errorMessage && (!required || (required && !value))}
              autoFocus={false}
            />
            <DeleteButton disabled={isDelButtonDisabled} onClick={() => handleDeleteRow(idx)}>
              <TrashIcon />
            </DeleteButton>
          </RowWrapper>
        ))}
      </ListContainer>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}

      <AddButton disabled={isAddButtonDisabled} variant='tertiary' onClick={handleAddRow}>
        <PlusIcon />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export default KeyValueInputsList;
