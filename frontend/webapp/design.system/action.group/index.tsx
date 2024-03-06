import React from 'react';
import { ActionItem } from '../action.item';

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

export const ActionsGroup: React.FC<ActionsListProps> = ({ actionGroups }) => {
  return (
    <>
      {actionGroups.map(
        (group, index) =>
          group.condition && <ActionItem key={index} {...group} />
      )}
    </>
  );
};
