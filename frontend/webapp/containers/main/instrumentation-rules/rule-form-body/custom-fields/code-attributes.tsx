import React, { useEffect, useMemo, useState } from 'react';
import styled, { css } from 'styled-components';
import { Checkbox, FieldError, FieldLabel } from '@/reuseable-components';
import { CodeAttributesType, type InstrumentationRuleInput } from '@/types';

type Props = {
  value: InstrumentationRuleInput;
  setValue: (key: keyof InstrumentationRuleInput, value: any) => void;
  formErrors: Record<string, string>;
};

type Parsed = InstrumentationRuleInput['codeAttributes'];

const ListContainer = styled.div<{ $hasError: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 8px;
  ${({ $hasError }) =>
    $hasError &&
    css`
      border: 1px solid ${({ theme }) => theme.text.error};
      border-radius: 16px;
      padding: 8px;
    `}
`;

const strictPicklist = [
  {
    id: CodeAttributesType.COLUMN,
    label: 'Collect code.column',
  },
  {
    id: CodeAttributesType.FILE_PATH,
    label: 'Collect code.filepath',
  },
  {
    id: CodeAttributesType.FUNCTION,
    label: 'Collect code.function',
  },
  {
    id: CodeAttributesType.LINE_NUMBER,
    label: 'Collect code.lineno',
  },
  {
    id: CodeAttributesType.NAMESPACE,
    label: 'Collect code.namespace',
  },
  {
    id: CodeAttributesType.STACKTRACE,
    label: 'Collect code.stacktrace',
  },
];

const CodeAttributes: React.FC<Props> = ({ value, setValue, formErrors }) => {
  const errorMessage = formErrors['codeAttributes'];

  const mappedValue = useMemo(
    () =>
      Object.entries(value['codeAttributes'] || {})
        .filter(([k, v]) => !!v)
        .map(([k]) => k),
    [value],
  );

  const [isLastSelection, setIsLastSelection] = useState(mappedValue.length === 1);

  useEffect(() => {
    if (!mappedValue.length) {
      const payload: Parsed = {
        [CodeAttributesType.COLUMN]: true,
        [CodeAttributesType.FILE_PATH]: true,
        [CodeAttributesType.FUNCTION]: true,
        [CodeAttributesType.LINE_NUMBER]: true,
        [CodeAttributesType.NAMESPACE]: true,
        [CodeAttributesType.STACKTRACE]: true,
      };

      setValue('codeAttributes', payload);
      setIsLastSelection(false);
    }
    // eslint-disable-next-line
  }, []);

  const handleChange = (id: string, isAdd: boolean) => {
    const arr = isAdd ? [...mappedValue, id] : mappedValue.filter((str) => str !== id);

    const payload: Parsed = {
      [CodeAttributesType.COLUMN]: arr.includes(CodeAttributesType.COLUMN) ? true : null,
      [CodeAttributesType.FILE_PATH]: arr.includes(CodeAttributesType.FILE_PATH) ? true : null,
      [CodeAttributesType.FUNCTION]: arr.includes(CodeAttributesType.FUNCTION) ? true : null,
      [CodeAttributesType.LINE_NUMBER]: arr.includes(CodeAttributesType.LINE_NUMBER) ? true : null,
      [CodeAttributesType.NAMESPACE]: arr.includes(CodeAttributesType.NAMESPACE) ? true : null,
      [CodeAttributesType.STACKTRACE]: arr.includes(CodeAttributesType.STACKTRACE) ? true : null,
    };

    setValue('codeAttributes', payload);
    setIsLastSelection(arr.length === 1);
  };

  return (
    <div>
      <FieldLabel title='Type of data to collect' required />
      <ListContainer $hasError={!!errorMessage}>
        {strictPicklist.map(({ id, label }) => (
          <Checkbox key={id} title={label} disabled={isLastSelection && mappedValue.includes(id)} value={mappedValue.includes(id)} onChange={(bool) => handleChange(id, bool)} />
        ))}
      </ListContainer>
      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </div>
  );
};

export default CodeAttributes;
