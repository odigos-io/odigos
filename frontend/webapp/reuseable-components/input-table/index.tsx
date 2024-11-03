import Image from 'next/image';
import { Text } from '../text';
import { Input } from '../input';
import { Button } from '../button';
import styled from 'styled-components';
import { FieldLabel } from '../field-label';
import React, { useState, useEffect, useRef } from 'react';

interface Props {
  columns: {
    title: string;
    keyName: string;
    type?: 'number';
    placeholder?: string;
    tooltip?: string;
    required?: boolean;
  }[];
  initialValues?: Record<string, any>[];
  value?: Record<string, any>[];
  onChange?: (values: Record<string, any>[]) => void;
}

const Container = styled.div`
  display: flex;
  flex-direction: column;
  width: 100%;
`;

const DeleteButton = styled.button`
  background: none;
  border: none;
  cursor: pointer;
  cursor: ${({ disabled }) => (disabled ? 'not-allowed' : 'pointer')};
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
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

export const InputTable: React.FC<Props> = ({ columns, initialValues = [], value = [], onChange }) => {
  const [initialObject, setInitialObject] = useState({});
  const [rows, setRows] = useState(value || initialValues);

  useEffect(() => {
    const init = {};
    columns.forEach(({ keyName }) => (init[keyName] = ''));
    setInitialObject(init);

    if (!rows.length) setRows([{ ...init }]);
  }, []);

  const recordedPairs = useRef('');

  useEffect(() => {
    // Filter out rows where any values are empty
    const validKeyValuePairs = rows.filter((row) => !Object.values(row).filter((val) => !val).length);
    const stringified = JSON.stringify(validKeyValuePairs);

    // Only trigger onChange if valid pairs have changed
    if (recordedPairs.current !== stringified) {
      recordedPairs.current = stringified;

      if (onChange) onChange(validKeyValuePairs);
    }
  }, [rows, onChange]);

  const handleAddRow = () => {
    setRows((prev) => {
      const payload = [...prev];
      payload.push({ ...initialObject });
      return payload;
    });
  };

  const handleDeleteRow = (idx: number) => {
    setRows((prev) => prev.filter((_, i) => i !== idx));
  };

  const handleChange = (key: string, val: string | number, idx: number) => {
    setRows((prev) => {
      const payload = [...prev];
      payload[idx][key] = val;
      return payload;
    });
  };

  // Check if any key or value field is empty
  const isAddButtonDisabled = rows.some((row) => !!Object.values(row).filter((val) => !val).length);
  const isDelButtonDisabled = rows.length <= 1;

  // adjust cell-width based on the amount of inputs on-screen,
  // the "0.4" is to consider the delete button
  const maxWidth = `${Math.floor(640 / (columns.length + 0.4))}px`;

  return (
    <Container>
      <table style={{ marginBottom: '12px', borderCollapse: 'collapse' }}>
        <thead>
          <tr>
            {columns.map(({ title, tooltip, required }) => (
              <th key={`input-table-head-${title}`} style={{ maxWidth, paddingLeft: 10 }}>
                <FieldLabel title={title} required={required} tooltip={tooltip} style={{ marginBottom: 0 }} />
              </th>
            ))}

            <th>{/* this is here because of delete button */}</th>
          </tr>
        </thead>

        <tbody>
          {rows.map((row, idx) => (
            <tr key={`input-table-row-${idx}`} style={{ height: '50px' }}>
              {columns.map(({ type, keyName, placeholder }, innerIdx) => (
                <td key={`input-table-${idx}-${keyName}`} style={{ maxWidth, padding: '0 2px' }}>
                  <Input
                    type={type}
                    placeholder={placeholder}
                    value={row[keyName]}
                    onChange={({ target: { value: val } }) => handleChange(keyName, type === 'number' ? Number(val) : val, idx)}
                    autoFocus={rows.length > 1 && idx === rows.length - 1 && innerIdx === 0}
                    style={{ maxWidth, paddingLeft: 10 }}
                  />
                </td>
              ))}

              <td>
                <DeleteButton disabled={isDelButtonDisabled} onClick={() => handleDeleteRow(idx)}>
                  <Image src='/icons/common/trash.svg' alt='Delete' width={16} height={16} />
                </DeleteButton>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      <AddButton disabled={isAddButtonDisabled} variant={'tertiary'} onClick={handleAddRow}>
        <Image src='/icons/common/plus.svg' alt='Add' width={16} height={16} />
        <ButtonText>ADD ENDPOINT FILTER</ButtonText>
      </AddButton>
    </Container>
  );
};

export default InputTable;
