import { KeyvalCheckbox, KeyvalText } from '@/design.system';
import React, { useState } from 'react';
import styled from 'styled-components';

interface CheckboxItem {
  id: string;
  label: string;
  checked: boolean;
}

interface MultiCheckboxProps {
  title?: string;
  checkboxes: CheckboxItem[];
  onSelectionChange: (selectedCheckboxes: CheckboxItem[]) => void;
}

const CheckboxWrapper = styled.div`
  display: flex;
  gap: 14px;
`;

export const MultiCheckboxComponent: React.FC<MultiCheckboxProps> = ({
  title,
  checkboxes,
  onSelectionChange,
}) => {
  const [selectedMonitors, setSelectedMonitors] =
    useState<CheckboxItem[]>(checkboxes);

  const handleCheckboxChange = (id: string) => {
    const updatedSelection = selectedMonitors.map((checkbox) => {
      if (checkbox.id === id) {
        return { ...checkbox, checked: !checkbox.checked };
      }
      return checkbox;
    });
    setSelectedMonitors(updatedSelection);
    onSelectionChange(updatedSelection);
  };

  return (
    <>
      {title && <KeyvalText size={14}>{title}</KeyvalText>}
      <CheckboxWrapper>
        {selectedMonitors.map((checkbox) => (
          <KeyvalCheckbox
            key={checkbox?.id}
            value={checkbox?.checked}
            onChange={() => handleCheckboxChange(checkbox?.id)}
            label={checkbox?.label}
          />
        ))}
      </CheckboxWrapper>
    </>
  );
};
