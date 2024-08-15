import React from 'react';
import styled from 'styled-components';
import { Text, Tag } from '@/reuseable-components';
import { MONITORS_OPTIONS } from '@/utils';
import Image from 'next/image';

interface MonitorButtonsProps {
  selectedMonitors: string[];
  onMonitorSelect: (monitor: string) => void;
}

const MonitorButtonsContainer = styled.div`
  display: flex;
  gap: 8px;
  margin-left: 12px;
`;

const MonitorsTitle = styled(Text)`
  opacity: 0.8;
  font-size: 14px;
  margin-left: 32px;
`;

const MonitorsTapList: React.FC<MonitorButtonsProps> = ({
  selectedMonitors,
  onMonitorSelect,
}) => {
  return (
    <>
      <MonitorsTitle>Monitor by:</MonitorsTitle>
      <MonitorButtonsContainer>
        {MONITORS_OPTIONS.map((monitor) => (
          <Tag
            id={monitor.id}
            isSelected={selectedMonitors.includes(monitor.id)}
            onClick={() => onMonitorSelect(monitor.id)}
          >
            <Image
              src={`/icons/monitors/${monitor.id}.svg`}
              alt="monitor"
              width={16}
              height={16}
            />
            <Text size={14}>{monitor.value}</Text>
          </Tag>
        ))}
      </MonitorButtonsContainer>
    </>
  );
};

export { MonitorsTapList };
