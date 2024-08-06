import React, { useState, useEffect } from 'react';
import styled from 'styled-components';
import { Checkbox } from '../checkbox';
import { Text } from '../text';

interface Monitor {
  id: string;
  title: string;
  tooltip?: string;
}

interface CheckboxListProps {
  monitors: Monitor[];
  title?: string;
}

const ListContainer = styled.div`
  display: flex;
  gap: 32px;
`;

const TextWrapper = styled.div`
  margin-bottom: 14px;
`;

const CheckboxList: React.FC<CheckboxListProps> = ({ monitors, title }) => {
  const [checkedState, setCheckedState] = useState<boolean[]>([]);

  useEffect(() => {
    // Initialize the checked state with all true if no initial values provided
    setCheckedState(Array(monitors.length).fill(true));
  }, [monitors.length]);

  const handleCheckboxChange = (index: number, value: boolean) => {
    const newCheckedState = [...checkedState];
    newCheckedState[index] = value;

    // Ensure at least one checkbox remains checked
    if (newCheckedState.filter((checked) => checked).length === 0) {
      return;
    }

    setCheckedState(newCheckedState);
  };

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
        {monitors.map((monitor, index) => (
          <Checkbox
            key={monitor.id}
            title={monitor.title}
            initialValue={checkedState[index]}
            onChange={(value) => handleCheckboxChange(index, value)}
            disabled={
              checkedState.filter((checked) => checked).length === 1 &&
              checkedState[index]
            }
          />
        ))}
      </ListContainer>
    </div>
  );
};

export { CheckboxList };
