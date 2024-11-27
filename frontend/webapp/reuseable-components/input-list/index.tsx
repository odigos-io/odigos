import Image from 'next/image';
import { Text } from '../text';
import { Input } from '../input';
import { Button } from '../button';
import styled from 'styled-components';
import { FieldLabel } from '../field-label';
import React, { useEffect, useMemo, useRef, useState } from 'react';

type Row = string;

interface InputListProps {
  initialValues?: Row[];
  value?: Row[];
  onChange: (values: Row[]) => void;
  title?: string;
  tooltip?: string;
  required?: boolean;
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

const INITIAL_ROW: Row = '';

const InputList: React.FC<InputListProps> = ({ initialValues = [], value, onChange, title, tooltip, required }) => {
  const [rows, setRows] = useState<Row[]>(value || initialValues);

  useEffect(() => {
    if (!rows.length) setRows([INITIAL_ROW]);
  }, []);

  // Filter out rows where either key or value is empty
  const validRows = useMemo(() => rows.filter((str) => !!str.trim()), [rows]);
  const recordedRows = useRef(JSON.stringify(validRows));

  useEffect(() => {
    const stringified = JSON.stringify(validRows);

    // Only trigger onChange if valid key-value pairs have changed
    if (recordedRows.current !== stringified) {
      recordedRows.current = stringified;

      if (onChange) onChange(validRows);
    }
  }, [validRows, onChange]);

  const handleAddInput = () => {
    setRows((prev) => {
      const payload = [...prev];
      payload.push(INITIAL_ROW);
      return payload;
    });
  };

  const handleDeleteInput = (idx: number) => {
    setRows((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleInputChange = (val: string, idx: number) => {
    setRows((prev) => {
      const payload = [...prev];
      payload[idx] = val;
      return payload;
    });
  };

  // Check if any input field is empty
  const isAddButtonDisabled = rows.some((input) => input.trim() === '');
  const isDelButtonDisabled = rows.length <= 1;

  return (
    <Container>
      <FieldLabel title={title} required={required} tooltip={tooltip} />

      {rows.map((val, idx) => (
        <InputRow key={`input-list-${idx}`}>
          <Input value={val} onChange={(e) => handleInputChange(e.target.value, idx)} autoFocus={!val && rows.length > 1 && idx === rows.length - 1} />
          <DeleteButton disabled={isDelButtonDisabled} onClick={() => handleDeleteInput(idx)}>
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
