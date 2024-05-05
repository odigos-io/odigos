import React from 'react';
import { ActionItem } from '@odigos-io/design-system';

interface Item {
  label: string;
  onClick: () => void;
  id: string;
  selected?: boolean;
  disabled?: boolean;
}

interface ActionItemProps {
  label: string;
  subTitle: string;
  items: Item[];
}

export const OdigosActionItem: React.FC<ActionItemProps> = (props) => {
  return <ActionItem {...props} />;
};
