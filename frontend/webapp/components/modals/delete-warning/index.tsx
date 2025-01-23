import React from 'react';
import { WarningModal } from '@/reuseable-components';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES } from '@/types';

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  name?: string;
  type?: OVERVIEW_ENTITY_TYPES;
  isLastItem?: boolean;
  onApprove: () => void;
  onDeny: () => void;
}

const DeleteWarning: React.FC<Props> = ({ isOpen, noOverlay, name, type, isLastItem, onApprove, onDeny }) => {
  const actionText = type === OVERVIEW_ENTITY_TYPES.SOURCE ? 'uninstrument' : 'delete';

  return (
    <WarningModal
      isOpen={isOpen}
      noOverlay={noOverlay}
      title={`${actionText.charAt(0).toUpperCase() + actionText.substring(1)}${name ? ` ${name}` : ''}`}
      description={`Are you sure you want to ${actionText}?`}
      note={
        isLastItem
          ? {
              type: NOTIFICATION_TYPE.WARNING,
              title: `You're about to ${actionText} the last ${type}`,
              message: '',
            }
          : undefined
      }
      approveButton={{
        text: 'Confirm',
        variant: 'danger',
        onClick: onApprove,
      }}
      denyButton={{
        text: 'Go Back',
        onClick: onDeny,
      }}
    />
  );
};

export { DeleteWarning };
