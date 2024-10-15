import React from 'react';
import { Text } from '../text';
import styled from 'styled-components';
import { Checkbox } from '../checkbox';
import { ExportedSignals } from '@/types';

interface Monitor {
  id: string;
  title: string;
  tooltip?: string;
}

interface CheckboxListProps {
  monitors: Monitor[];
  title?: string;
  exportedSignals: ExportedSignals;
  handleSignalChange: (signal: string, value: boolean) => void;
}

const ListContainer = styled.div`
  display: flex;
  gap: 32px;
`;

const TextWrapper = styled.div`
  margin-bottom: 14px;
`;

const CheckboxList: React.FC<CheckboxListProps> = ({
  monitors,
  title,
  exportedSignals,
  handleSignalChange,
}) => {
  function isItemDisabled(item: Monitor) {
    const selectedItems = Object.values(exportedSignals).filter(
      (value) => value
    );

    const trueValues = Object.values(exportedSignals).filter(Boolean);

    return (
      (monitors.length === 1 && trueValues.length === 1) ||
      (selectedItems.length === 1 && exportedSignals[item.id])
    );
  }

  return (
    <div>
      {title && (
        <TextWrapper>
          <Text size={14} weight={300} opacity={0.8}>
            {title}
          </Text>
        </TextWrapper>
      )}
      <ListContainer>
        {monitors.map((monitor) => (
          <Checkbox
            key={monitor.id}
            title={monitor.title}
            initialValue={exportedSignals[monitor.id]}
            onChange={(value) => handleSignalChange(monitor.id, value)}
            disabled={isItemDisabled(monitor)}
          />
        ))}
      </ListContainer>
    </div>
  );
};

export { CheckboxList };
