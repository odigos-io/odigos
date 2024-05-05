import { ActionsGroup } from '@odigos-io/design-system';
import React from 'react';

// Define the type for individual action items
interface ActionItem {
  label: string;
  onClick: () => void;
  id: string;
  selected?: boolean;
  disabled?: boolean;
}

// Define the type for the groups of action items, including any conditional rendering logic
interface ActionGroup {
  label: string;
  subTitle: string;
  items: ActionItem[];
  condition?: boolean; // Optional condition to determine if the group should be rendered
}

// Props for the container component that will render the list of action groups
interface ActionsListProps {
  actionGroups: ActionGroup[];
}

export const OdigosActionsGroup: React.FC<ActionsListProps> = ({
  actionGroups,
}) => {
  return <ActionsGroup actionGroups={actionGroups} />;
};
