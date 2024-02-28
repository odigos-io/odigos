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
  const [isCheckboxDisabled, setIsCheckboxDisabled] = useState<boolean>(false);
  const [selectedMonitors, setSelectedMonitors] =
    useState<CheckboxItem[]>(checkboxes);

  const handleCheckboxChange = (id: string) => {
    // Calculate the number of currently checked checkboxes
    const currentlyCheckedCount = selectedMonitors.filter(
      (checkbox) => checkbox.checked
    ).length;

    // Update logic to ensure at least one checkbox remains checked
    const updatedSelection = selectedMonitors.map((checkbox) => {
      if (checkbox.id === id) {
        // Prevent unchecking if this is the last checked checkbox
        if (checkbox.checked && currentlyCheckedCount === 1) {
          return checkbox; // Do not change the state if attempting to uncheck the last checked checkbox
        }

        return { ...checkbox, checked: !checkbox.checked };
      }
      return checkbox;
    });

    setSelectedMonitors(updatedSelection);
    onSelectionChange(updatedSelection);

    const newCheckedCount = updatedSelection.filter(
      (checkbox) => checkbox.checked
    ).length;
    setIsCheckboxDisabled(newCheckedCount <= 1);
  };

  return (
    <>
      {title && (
        <KeyvalText size={14} weight={600}>
          {title}
        </KeyvalText>
      )}
      <CheckboxWrapper>
        {selectedMonitors.map((checkbox) => (
          <KeyvalCheckbox
            disabled={isCheckboxDisabled && checkbox.checked}
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
