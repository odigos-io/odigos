import { Text } from '../text';
import { Checkbox } from '../checkbox';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';
import { MONITORING_OPTIONS, SignalLowercase, SignalUppercase } from '@/utils';

interface MonitoringCheckboxesProps {
  isVertical?: boolean;
  allowedSignals?: SignalUppercase[];
  selectedSignals: SignalUppercase[];
  setSelectedSignals: (value: SignalUppercase[]) => void;
}

const ListContainer = styled.div<{ isVertical?: boolean }>`
  display: flex;
  flex-direction: ${({ isVertical }) => (isVertical ? 'column' : 'row')};
  gap: ${({ isVertical }) => (isVertical ? '16px' : '32px')};
`;

const TextWrapper = styled.div`
  margin-bottom: 14px;
`;

const monitors = MONITORING_OPTIONS;

const isAllowed = (type: SignalLowercase, allowedSignals: MonitoringCheckboxesProps['allowedSignals']) => {
  return !allowedSignals?.length || !!allowedSignals?.find((str) => str === type.toUpperCase());
};

const isSelected = (type: SignalLowercase, selectedSignals: MonitoringCheckboxesProps['selectedSignals']) => {
  return !!selectedSignals?.find((str) => str === type.toUpperCase());
};

const MonitoringCheckboxes: React.FC<MonitoringCheckboxesProps> = ({ isVertical, allowedSignals, selectedSignals, setSelectedSignals }) => {
  const [isLastSelection, setIsLastSelection] = useState(false);

  useEffect(() => {
    const payload: SignalUppercase[] = [];

    monitors.forEach(({ type }) => {
      if (isAllowed(type, allowedSignals)) {
        payload.push(type.toUpperCase() as SignalUppercase);
      }
    });

    setSelectedSignals(payload);
    setIsLastSelection(payload.length === 1);
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
      <TextWrapper>
        <Text>Monitoring</Text>
      </TextWrapper>

      <ListContainer isVertical={isVertical}>
        {monitors.map((monitor) => {
          const allowed = isAllowed(monitor.type, allowedSignals);
          const selected = isSelected(monitor.type, selectedSignals);

          if (!allowed) return null;

          return (
            <Checkbox
              key={monitor.id}
              title={monitor.title}
              disabled={!allowed || (isLastSelection && selected)}
              initialValue={selected}
              onChange={(value) => handleChange(monitor.type, value)}
            />
          );
        })}
      </ListContainer>
    </div>
  );
};

export { MonitoringCheckboxes };
