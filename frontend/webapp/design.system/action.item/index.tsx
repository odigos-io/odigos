import { Check, Expand } from '@/assets/icons/app';
import { KeyvalText } from '@/design.system';
import { useOnClickOutside } from '@/hooks';
import React, { useRef, useState } from 'react';
import styled from 'styled-components';

// Styled components
const Label = styled.label`
  cursor: pointer;
  display: flex;
  gap: 4px;
  p {
    color: ${({ theme }) => theme.colors.light_grey};
    &:hover {
      color: ${({ theme }) => theme.colors.white};
    }
  }
`;

const Popup = styled.div<{ isOpen: boolean }>`
  display: ${(props: { isOpen: boolean }) => (props.isOpen ? 'block' : 'none')};
  position: absolute;
  right: 0px;
  box-shadow: 0px 8px 16px 0px rgba(0, 0, 0, 0.2);
  z-index: 9999;
  flex-direction: column;
  border-radius: 8px;
  border: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  background: ${({ theme }) => theme.colors.dark};
  margin-top: 5px;
`;

const PopupItem = styled.div<{ disabled: boolean }>`
  display: flex;
  padding: 7px 12px;
  gap: 4px;
  border-top: ${({ theme }) => `1px solid ${theme.colors.blue_grey}`};
  align-items: center;
  opacity: ${({ disabled }) => (disabled ? 0.5 : 1)};
  pointer-events: ${({ disabled }) => (disabled ? 'none' : 'auto')};
  cursor: pointer;
  p {
    cursor: pointer !important;
  }

  &:hover {
    background: ${({ theme }) => theme.colors.light_dark};
  }
`;

interface Item {
  label: string;
  onClick: () => void;
  id: string;
  selected?: boolean;
  disabled?: boolean;
}

interface DropdownProps {
  label: string;
  subTitle: string;
  items: Item[];
}

export const ActionItem: React.FC<DropdownProps> = ({
  label,
  items,
  subTitle,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const ref = useRef(null);
  useOnClickOutside(ref, () => setIsOpen(false));

  return (
    <div ref={ref} style={{ position: 'relative' }}>
      <Label onClick={() => setIsOpen(!isOpen)}>
        <KeyvalText size={12} weight={600}>
          {label}
        </KeyvalText>
        <Expand />
      </Label>
      <Popup isOpen={isOpen}>
        <div style={{ padding: 12, width: 120 }}>
          <KeyvalText size={12} weight={600}>
            {subTitle}
          </KeyvalText>
        </div>
        {items.map((item, index) => (
          <PopupItem
            key={index}
            onClick={item.onClick}
            disabled={!!item.disabled}
          >
            {item.selected ? <Check /> : <div style={{ width: 10 }} />}
            <KeyvalText size={12} weight={600}>
              {item.label}
            </KeyvalText>
          </PopupItem>
        ))}
      </Popup>
    </div>
  );
};
