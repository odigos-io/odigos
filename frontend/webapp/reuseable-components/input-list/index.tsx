import React, { type KeyboardEventHandler, useEffect, useMemo, useRef, useState } from 'react';
import styled from 'styled-components';
import { PlusIcon, TrashIcon } from '@/assets';
import { Button, FieldError, FieldLabel, Input, Text } from '@/reuseable-components';

type Row = string;

interface InputListProps {
  initialValues?: Row[];
  value?: Row[];
  onChange: (values: Row[]) => void;
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

const InputList: React.FC<InputListProps> = ({ initialValues = [], value, onChange, title, tooltip, required, errorMessage }) => {
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

      <ListContainer>
        {rows.map((val, idx) => (
          <RowWrapper key={`input-list-${idx}`}>
            <Input value={val} onChange={(e) => handleInputChange(e.target.value, idx)} hasError={!!errorMessage} autoFocus={!val && rows.length > 1 && idx === rows.length - 1} />
            <DeleteButton disabled={isDelButtonDisabled} onClick={() => handleDeleteInput(idx)}>
              <TrashIcon />
            </DeleteButton>
          </RowWrapper>
        ))}
      </ListContainer>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}

      <AddButton disabled={isAddButtonDisabled} variant='tertiary' onClick={handleAddInput}>
        <PlusIcon />
        <ButtonText>ADD ATTRIBUTE</ButtonText>
      </AddButton>
    </Container>
  );
};

export { InputList };
