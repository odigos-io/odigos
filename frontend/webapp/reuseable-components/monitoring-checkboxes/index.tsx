import React, { useEffect, useRef, useState } from 'react';
import styled, { css } from 'styled-components';
import { Checkbox, FieldError, FieldLabel } from '@/reuseable-components';
import { MONITORS_OPTIONS, SignalLowercase, SignalUppercase } from '@/utils';

interface Props {
  isVertical?: boolean;
  title?: string;
  required?: boolean;
  errorMessage?: string;
  allowedSignals?: SignalUppercase[];
  selectedSignals: SignalUppercase[];
  setSelectedSignals: (value: SignalUppercase[]) => void;
}

const ListContainer = styled.div<{ $isVertical?: Props['isVertical']; $hasError: boolean }>`
  display: flex;
  flex-direction: ${({ $isVertical }) => ($isVertical ? 'column' : 'row')};
  gap: ${({ $isVertical }) => ($isVertical ? '12px' : '24px')};
  ${({ $hasError }) =>
    $hasError &&
    css`
      border: 1px solid ${({ theme }) => theme.text.error};
      border-radius: 32px;
      padding: 8px;
    `}
`;

const monitors = MONITORS_OPTIONS;

const isAllowed = (type: SignalLowercase, allowedSignals: Props['allowedSignals']) => {
  return !allowedSignals?.length || !!allowedSignals?.find((str) => str === type.toUpperCase());
};

const isSelected = (type: SignalLowercase, selectedSignals: Props['selectedSignals']) => {
  return !!selectedSignals?.find((str) => str === type.toUpperCase());
};

export const MonitoringCheckboxes: React.FC<Props> = ({ isVertical, title = 'Monitoring', required, errorMessage, allowedSignals, selectedSignals, setSelectedSignals }) => {
  const [isLastSelection, setIsLastSelection] = useState(selectedSignals.length === 1);
  const recordedRows = useRef(JSON.stringify(selectedSignals));

  useEffect(() => {
    const payload: SignalUppercase[] = selectedSignals;

    if (!payload.length) {
      monitors.forEach(({ id }) => {
        if (isAllowed(id, allowedSignals)) {
          payload.push(id.toUpperCase() as SignalUppercase);
        }
      });
    }

    const stringified = JSON.stringify(payload);

    if (recordedRows.current !== stringified) {
      recordedRows.current = stringified;
      setSelectedSignals(payload);
      setIsLastSelection(payload.length === 1);
    }

    return () => {
      recordedRows.current = '';
    };
    // eslint-disable-next-line
  }, [allowedSignals]);

  const handleChange = (key: SignalLowercase, isAdd: boolean) => {
    const keyUpper = key.toUpperCase() as SignalUppercase;
    const payload = isAdd ? [...selectedSignals, keyUpper] : selectedSignals.filter((str) => str !== keyUpper);

    setSelectedSignals(payload);
    setIsLastSelection(payload.length === 1);
  };

  return (
    <div>
      {title && <FieldLabel title={title} required={required} />}

      <ListContainer $isVertical={isVertical} $hasError={!!errorMessage}>
        {monitors.map((monitor) => {
          const allowed = isAllowed(monitor.id, allowedSignals);
          const selected = isSelected(monitor.id, selectedSignals);

          if (!allowed) return null;

          return <Checkbox key={monitor.id} title={monitor.value} disabled={!allowed || (isLastSelection && selected)} value={selected} onChange={(value) => handleChange(monitor.id, value)} />;
        })}
      </ListContainer>

      {!!errorMessage && <FieldError>{errorMessage}</FieldError>}
    </div>
  );
};
