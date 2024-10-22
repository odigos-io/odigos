import { Text } from '../text';
import { Checkbox } from '../checkbox';
import styled from 'styled-components';
import React, { useEffect, useState } from 'react';
import { MONITORING_OPTIONS, SignalLowercase, SignalUppercase } from '@/utils';

interface MonitoringCheckboxesProps {
  isVertical?: boolean;
  allowedSignals?: (SignalUppercase | SignalLowercase)[];
  selectedSignals: (SignalUppercase | SignalLowercase)[];
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
const initialStatuses: Record<SignalLowercase, boolean> = {
  logs: false,
  metrics: false,
  traces: false,
};

const MonitoringCheckboxes: React.FC<MonitoringCheckboxesProps> = ({ isVertical, allowedSignals, selectedSignals, setSelectedSignals }) => {
  const [signalStatuses, setSignalStatuses] = useState({ ...initialStatuses });

  useEffect(() => {
    const payload = { ...initialStatuses };

    selectedSignals.forEach((str) => {
      payload[str.toLowerCase()] = true;
    });

    if (JSON.stringify(payload) !== JSON.stringify(signalStatuses)) {
      setSignalStatuses(payload);
    }
  }, [selectedSignals]);

  const handleChange = (key: keyof typeof signalStatuses, value: boolean) => {
    const selected: SignalUppercase[] = [];

    setSignalStatuses((prev) => {
      const payload = { ...prev, [key]: value };

      Object.entries(payload).forEach(([sig, bool]) => {
        if (bool) selected.push(sig.toUpperCase() as SignalUppercase);
      });

      return payload;
    });

    setSelectedSignals(selected);
  };

  const isDisabled = (item: (typeof MONITORING_OPTIONS)[0]) => {
    return !!allowedSignals && !allowedSignals.find((str) => str.toLowerCase() === item.type);
  };

  return (
    <div>
      <TextWrapper>
        <Text>Monitoring</Text>
      </TextWrapper>

      <ListContainer isVertical={isVertical}>
        {monitors.map((monitor) => (
          <Checkbox
            key={monitor.id}
            title={monitor.title}
            initialValue={signalStatuses[monitor.type]}
            onChange={(value) => handleChange(monitor.type, value)}
            disabled={isDisabled(monitor)}
          />
        ))}
      </ListContainer>
    </div>
  );
};

export { MonitoringCheckboxes };
