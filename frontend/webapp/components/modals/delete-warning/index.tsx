import React from 'react';
import { NOTIFICATION_TYPE } from '@/types';
import { Types, WarningModal } from '@odigos/ui-components';

interface Props {
  isOpen: boolean;
  noOverlay?: boolean;
  name?: string;
  type?: Types.ENTITY_TYPES;
  isLastItem?: boolean;
  onApprove: () => void;
  onDeny: () => void;
}

const DeleteWarning: React.FC<Props> = ({ isOpen, noOverlay, name, type, isLastItem, onApprove, onDeny }) => {
  const actionText = type === Types.ENTITY_TYPES.SOURCE ? 'uninstrument' : 'delete';

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
